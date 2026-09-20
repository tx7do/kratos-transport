package websocket

import (
	"net/url"
	"sync"
	"time"

	ws "github.com/gorilla/websocket"
	"github.com/tx7do/go-utils/id"
)

var channelBufSize = 256

type SessionID string

type Session struct {
	id SessionID

	conn    *ws.Conn
	queries url.Values

	send chan []byte

	hooks SessionHooks

	// readTimeout 空闲读超时；0 表示不设置（默认）。由 Server 的 WithTimeout 接线，
	// 用于回收半开连接；每收到一条消息自动顺延
	readTimeout time.Duration

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

	getPayloadType() PayloadType
}

func NewSession(hooks SessionHooks, conn *ws.Conn, vars url.Values) *Session {
	if conn == nil {
		panic("conn cannot be nil")
	}

	c := &Session{
		id:      SessionID(id.NewGUIDv4(false)),
		conn:    conn,
		queries: vars,
		send:    make(chan []byte, channelBufSize),
		hooks:   hooks,
		done:    make(chan struct{}),
	}

	return c
}

func (s *Session) Conn() *ws.Conn {
	s.connMu.RLock()
	defer s.connMu.RUnlock()

	return s.conn
}

func (s *Session) Queries() url.Values {
	return s.queries
}

func (s *Session) SessionID() SessionID {
	return s.id
}

func (s *Session) SendMessage(message []byte) {
	select {
	case <-s.done:
		return
	case s.send <- message:
	case <-time.After(5 * time.Second):
		// 慢消费者（writePump 阻塞）不能无限拖住调用方/广播方
		LogErrorf("session %s send buffer full, message dropped", s.SessionID())
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

func (s *Session) sendPingMessage(message string) error {
	s.connMu.RLock()
	conn := s.conn
	s.connMu.RUnlock()
	if conn == nil {
		return nil
	}
	return conn.WriteControl(ws.PingMessage, []byte(message), time.Now().Add(5*time.Second))
}

func (s *Session) sendPongMessage(message string) error {
	s.connMu.RLock()
	conn := s.conn
	s.connMu.RUnlock()
	if conn == nil {
		return nil
	}
	// gorilla 单写者约束：控制帧必须走 WriteControl，与 writePump 的数据帧并发安全
	return conn.WriteControl(ws.PongMessage, []byte(message), time.Now().Add(5*time.Second))
}

func (s *Session) sendTextMessage(message string) error {
	s.connMu.RLock()
	conn := s.conn
	s.connMu.RUnlock()
	if conn == nil {
		return nil
	}
	return conn.WriteMessage(ws.TextMessage, []byte(message))
}

func (s *Session) sendBinaryMessage(message []byte) error {
	s.connMu.RLock()
	conn := s.conn
	s.connMu.RUnlock()
	if conn == nil {
		return nil
	}
	return conn.WriteMessage(ws.BinaryMessage, message)
}

func (s *Session) writePump() {
	defer s.wg.Done()
	defer s.Close()

	for {
		select {
		case <-s.done:
			return

		case msg := <-s.send:

			var err error
			switch s.hooks.getPayloadType() {
			case PayloadTypeBinary:
				if err = s.sendBinaryMessage(msg); err != nil {
					LogError("write binary message error: ", err)
					return
				}
				break

			case PayloadTypeText:
				if err = s.sendTextMessage(string(msg)); err != nil {
					LogError("write text message error: ", err)
					return
				}
				break
			}

		}
	}
}

func (s *Session) readPump() {
	defer s.wg.Done()
	defer s.Close()

	conn := s.Conn()
	if s.readTimeout > 0 {
		_ = conn.SetReadDeadline(time.Now().Add(s.readTimeout))
		// gorilla 在内部消费 Ping/Pong 控制帧且不使 ReadMessage 返回，
		// 必须用 PongHandler 顺延 deadline，否则仅心跳保活的连接会被误杀
		conn.SetPongHandler(func(string) error {
			return conn.SetReadDeadline(time.Now().Add(s.readTimeout))
		})
	}

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

		if s.readTimeout > 0 {
			_ = conn.SetReadDeadline(time.Now().Add(s.readTimeout))
		}

		messageType, data, err := conn.ReadMessage()
		if err != nil {
			if ws.IsUnexpectedCloseError(err, ws.CloseNormalClosure, ws.CloseGoingAway, ws.CloseAbnormalClosure) {
				LogErrorf("read message error: %v", err)
			}
			return
		}

		switch messageType {
		case ws.CloseMessage:
			return

		case ws.BinaryMessage:
			_ = s.hooks.handleSocketRawData(s.SessionID(), data)
			break

		case ws.TextMessage:
			_ = s.hooks.handleSocketRawData(s.SessionID(), data)
			break

		case ws.PingMessage:
			if err = s.sendPongMessage(""); err != nil {
				LogError("write pong message error: ", err)
				return
			}
			break

		case ws.PongMessage:
			break
		}

	}
}
