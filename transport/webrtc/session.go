package webrtc

import (
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/tx7do/go-utils/id"
)

var channelBufSize = 256

type SessionID string

type Session struct {
	id SessionID

	pc      *webrtc.PeerConnection
	dc      *webrtc.DataChannel
	queries url.Values

	send  chan []byte
	hooks SessionHooks

	lastReadMessageTime  time.Time // Last time the session received application data.
	lastWriteMessageTime time.Time // Last time the session sent application data.

	connMu     sync.RWMutex
	done       chan struct{}
	listenOnce sync.Once
	closeOnce  sync.Once
	openOnce   sync.Once
	wg         sync.WaitGroup
}

// SessionHooks defines callbacks used by Session to interact with upper-layer server logic.
type SessionHooks interface {
	removeSession(*Session)

	handleSocketRawData(SessionID, []byte) error

	getPayloadType() PayloadType
}

func NewSession(hooks SessionHooks, pc *webrtc.PeerConnection, vars url.Values) *Session {
	if pc == nil {
		panic("peer connection cannot be nil")
	}

	return &Session{
		id:      SessionID(id.NewGUIDv4(false)),
		pc:      pc,
		queries: vars,
		send:    make(chan []byte, channelBufSize),
		hooks:   hooks,
		done:    make(chan struct{}),
	}
}

func (s *Session) PeerConnection() *webrtc.PeerConnection {
	s.connMu.RLock()
	defer s.connMu.RUnlock()

	return s.pc
}

func (s *Session) DataChannel() *webrtc.DataChannel {
	s.connMu.RLock()
	defer s.connMu.RUnlock()

	return s.dc
}

func (s *Session) Queries() url.Values {
	return s.queries
}

func (s *Session) SessionID() SessionID {
	return s.id
}

func (s *Session) BindDataChannel(dc *webrtc.DataChannel, onOpen func()) {
	if dc == nil {
		return
	}

	s.connMu.Lock()
	s.dc = dc
	s.connMu.Unlock()

	dc.OnOpen(func() {
		s.openOnce.Do(func() {
			if onOpen != nil {
				onOpen()
			}
		})
	})

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		s.lastReadMessageTime = time.Now()
		if s.hooks == nil {
			return
		}
		if err := s.hooks.handleSocketRawData(s.SessionID(), msg.Data); err != nil {
			LogError("handle socket raw data error:", err)
		}
	})

	dc.OnClose(func() {
		s.Close()
	})

	dc.OnError(func(err error) {
		if err != nil && !isExpectedDataChannelCloseError(err) {
			LogError("data channel error:", err)
		}
		s.Close()
	})
}

func isExpectedDataChannelCloseError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "user initiated abort") ||
		strings.Contains(msg, "closing") ||
		strings.Contains(msg, "closed")
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
		s.closeConnection()

		if s.hooks != nil {
			s.hooks.removeSession(s)
		}
	})
}

func (s *Session) Listen() {
	s.listenOnce.Do(func() {
		s.wg.Add(1)
		go s.writePump()
	})
}

func (s *Session) Wait() {
	s.wg.Wait()
}

func (s *Session) closeConnection() {
	s.connMu.Lock()
	dc := s.dc
	pc := s.pc
	s.dc = nil
	s.pc = nil
	s.connMu.Unlock()

	if dc != nil {
		if err := dc.Close(); err != nil {
			LogDebugf("close data channel error: %s", err)
		}
	}

	if pc != nil {
		if err := pc.Close(); err != nil {
			LogDebugf("close peer connection error: %s", err)
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
			s.lastWriteMessageTime = time.Now()

			dc := s.DataChannel()
			if dc == nil {
				return
			}

			var err error
			switch s.hooks.getPayloadType() {
			case PayloadTypeText:
				err = dc.SendText(string(msg))
			default:
				err = dc.Send(msg)
			}
			if err != nil {
				LogError("write data channel message error:", err)
				return
			}
		}
	}
}
