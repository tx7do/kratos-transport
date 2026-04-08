package tcp

import (
	"net"
	"sync"

	"github.com/tx7do/go-utils/id"
)

var channelBufSize = 256
var recvBufferSize = 256000

type SessionID string

type Session struct {
	id SessionID

	conn  net.Conn
	hooks SessionHooks

	send chan []byte

	connMu     sync.RWMutex
	done       chan struct{}
	listenOnce sync.Once
	closeOnce  sync.Once
	wg         sync.WaitGroup
}

// SessionHooks defines callbacks used by Session to interact with upper-layer server logic.
type SessionHooks interface {
	removeSession(*Session)
	handleSocketRawData(SessionID, []byte) error
}

func NewSession(conn net.Conn, hooks SessionHooks) *Session {
	if conn == nil {
		panic("conn cannot be nil")
	}

	c := &Session{
		id:    SessionID(id.NewGUIDv4(false)),
		conn:  conn,
		done:  make(chan struct{}),
		send:  make(chan []byte, channelBufSize),
		hooks: hooks,
	}

	return c
}

func (s *Session) Conn() net.Conn {
	s.connMu.RLock()
	defer s.connMu.RUnlock()

	return s.conn
}

func (s *Session) SessionID() SessionID {
	return s.id
}

func (s *Session) SendMessage(message []byte) {
	select {
	case <-s.done:
		return
	case s.send <- message:
	}
}

func (s *Session) Close() {
	s.closeOnce.Do(func() {
		close(s.done)
		s.closeConnect()

		if s.hooks != nil {
			s.hooks.removeSession(s)
		}
	})
}

func (s *Session) Listen() {
	s.listenOnce.Do(func() {
		s.wg.Add(2)

		go s.writePump()
		go s.readPump()
	})
}

func (s *Session) Wait() {
	s.wg.Wait()
}

func (s *Session) closeConnect() {
	//LogInfo(c.SessionID(), " connection closed")
	s.connMu.Lock()
	conn := s.conn
	s.conn = nil
	s.connMu.Unlock()

	if conn != nil {
		if err := conn.Close(); err != nil {
			LogErrorf("disconnect error: %s", err.Error())
		}
	}
}

func (s *Session) writePump() {
	defer s.wg.Done()
	defer s.Close()

	for {
		select {
		case <-s.done:
			return

		case msg := <-s.send:
			conn := s.Conn()
			if conn == nil {
				return
			}
			var err error
			if _, err = conn.Write(msg); err != nil {
				select {
				case <-s.done:
					return
				default:
				}
				LogError("write message error: ", err)
				return
			}
		}
	}
}

func (s *Session) readPump() {
	defer s.wg.Done()
	defer s.Close()

	buf := make([]byte, recvBufferSize)
	var err error
	var readLen int

	for {
		select {
		case <-s.done:
			return
		default:
		}

		conn := s.Conn()
		if conn == nil {
			return
		}

		if readLen, err = conn.Read(buf); err != nil {
			select {
			case <-s.done:
				return
			default:
			}
			LogErrorf("read message error: %v", err)
			return
		}

		if s.hooks == nil {
			continue
		}

		if err = s.hooks.handleSocketRawData(s.SessionID(), buf[:readLen]); err != nil {
			LogErrorf("process message error: %v", err)
		}
	}
}
