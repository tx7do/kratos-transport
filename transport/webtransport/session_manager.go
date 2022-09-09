package webtransport

import (
	"context"
	"sync"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/http3"
	"github.com/lucas-clemente/quic-go/quicvarint"
)

// session is the map value in the sessions map
type session struct {
	created chan struct{} // is closed once the session map has been initialized
	counter int           // how many streams are waiting for this session to be established
	conn    *Session
}

type sessionMap map[SessionID]*session
type connectMap map[http3.StreamCreator]sessionMap

type sessionManager struct {
	refCount  sync.WaitGroup
	ctx       context.Context
	ctxCancel context.CancelFunc

	timeout time.Duration

	mx          sync.Mutex
	connections connectMap
}

func newSessionManager(timeout time.Duration) *sessionManager {
	m := &sessionManager{
		timeout:     timeout,
		connections: make(connectMap),
	}
	m.ctx, m.ctxCancel = context.WithCancel(context.Background())
	return m
}

// AddStream adds a new bidirectional qStream to a WebTransport session.
// If the WebTransport session has not yet been established,
// it starts a new go routine and waits for establishment of the session.
// If that takes longer than timeout, the qStream is reset.
func (m *sessionManager) AddStream(qConn http3.StreamCreator, qStream quic.Stream, id SessionID) {
	sess, isExisting := m.getOrCreateSession(qConn, id)
	if isExisting {
		sess.conn.addIncomingStream(qStream)
		return
	}

	m.refCount.Add(1)
	go func() {
		defer m.refCount.Done()
		m.handleStream(qStream, sess)

		m.mx.Lock()
		defer m.mx.Unlock()

		sess.counter--
		// Once no more streams are waiting for this session to be established,
		// and this session is still outstanding, delete it from the map.
		if sess.counter == 0 && sess.conn == nil {
			m.maybeDelete(qConn, id)
		}
	}()
}

func (m *sessionManager) maybeDelete(qConn http3.StreamCreator, id SessionID) {
	sessions, ok := m.connections[qConn]
	if !ok { // should never happen
		return
	}
	delete(sessions, id)
	if len(sessions) == 0 {
		delete(m.connections, qConn)
	}
}

// AddUniStream adds a new unidirectional qStream to a WebTransport session.
// If the WebTransport session has not yet been established,
// it starts a new go routine and waits for establishment of the session.
// If that takes longer than timeout, the qStream is reset.
func (m *sessionManager) AddUniStream(qConn http3.StreamCreator, qStream quic.ReceiveStream) {
	idv, err := quicvarint.Read(quicvarint.NewReader(qStream))
	if err != nil {
		qStream.CancelRead(1337)
	}
	id := SessionID(idv)

	sess, isExisting := m.getOrCreateSession(qConn, id)
	if isExisting {
		sess.conn.addIncomingUniStream(qStream)
		return
	}

	m.refCount.Add(1)
	go func() {
		defer m.refCount.Done()
		m.handleUniStream(qStream, sess)

		m.mx.Lock()
		defer m.mx.Unlock()

		sess.counter--
		// Once no more streams are waiting for this session to be established,
		// and this session is still outstanding, delete it from the map.
		if sess.counter == 0 && sess.conn == nil {
			m.maybeDelete(qConn, id)
		}
	}()
}

func (m *sessionManager) getOrCreateSession(qConn http3.StreamCreator, id SessionID) (sess *session, existed bool) {
	m.mx.Lock()
	defer m.mx.Unlock()

	sessions, ok := m.connections[qConn]
	if !ok {
		sessions = make(sessionMap)
		m.connections[qConn] = sessions
	}

	sess, ok = sessions[id]
	if ok && sess.conn != nil {
		return sess, true
	}
	if !ok {
		sess = &session{created: make(chan struct{})}
		sessions[id] = sess
	}
	sess.counter++
	return sess, false
}

func (m *sessionManager) handleStream(qStream quic.Stream, sess *session) {
	t := time.NewTimer(m.timeout)
	defer t.Stop()

	// When multiple streams are waiting for the same session to be established,
	// the timeout is calculated for every qStream separately.
	select {
	case <-sess.created:
		sess.conn.addIncomingStream(qStream)
	case <-t.C:
		qStream.CancelRead(WebTransportBufferedStreamRejectedErrorCode)
		qStream.CancelWrite(WebTransportBufferedStreamRejectedErrorCode)
	case <-m.ctx.Done():
	}
}

func (m *sessionManager) handleUniStream(qStream quic.ReceiveStream, sess *session) {
	t := time.NewTimer(m.timeout)
	defer t.Stop()

	// When multiple streams are waiting for the same session to be established,
	// the timeout is calculated for every qStream separately.
	select {
	case <-sess.created:
		sess.conn.addIncomingUniStream(qStream)
	case <-t.C:
		qStream.CancelRead(WebTransportBufferedStreamRejectedErrorCode)
	case <-m.ctx.Done():
	}
}

// AddSession add a new WebTransport session.
func (m *sessionManager) AddSession(qConn http3.StreamCreator, id SessionID, requestStream quic.Stream) *Session {
	conn := newSession(id, qConn, requestStream)

	m.mx.Lock()
	defer m.mx.Unlock()

	sessions, ok := m.connections[qConn]
	if !ok {
		sessions = make(sessionMap)
		m.connections[qConn] = sessions
	}
	if sess, ok := sessions[id]; ok {
		// We might already have an entry of this session.
		// This can happen when we receive a qStream for this WebTransport session before we complete the HTTP request
		// that establishes the session.
		sess.conn = conn
		close(sess.created)
		return conn
	}
	c := make(chan struct{})
	close(c)
	sessions[id] = &session{created: c, conn: conn}
	return conn
}

func (m *sessionManager) Close() {
	m.ctxCancel()
	m.refCount.Wait()
}
