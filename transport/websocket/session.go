package websocket

import (
	"net/url"
	"time"

	"github.com/google/uuid"
	ws "github.com/gorilla/websocket"
)

var channelBufSize = 256

type SessionID string

type Session struct {
	id      SessionID
	conn    *ws.Conn
	queries url.Values
	send    chan []byte
	server  *Server

	lastReadMessageTime  time.Time // 最后一次读取消息的时间
	lastWriteMessageTime time.Time // 最后一次发送消息的时间
}

func NewSession(server *Server, conn *ws.Conn, vars url.Values) *Session {
	if conn == nil {
		panic("conn cannot be nil")
	}

	u1, _ := uuid.NewUUID()

	c := &Session{
		id:      SessionID(u1.String()),
		conn:    conn,
		queries: vars,
		send:    make(chan []byte, channelBufSize),
		server:  server,
	}

	return c
}

func (s *Session) Conn() *ws.Conn {
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
	case s.send <- message:
	}
}

func (s *Session) Close() {
	s.server.unregister <- s
	s.closeConnect()
}

func (s *Session) Listen() {
	go s.writePump()
	go s.readPump()
}

func (s *Session) closeConnect() {
	//LogInfo(s.SessionID(), " connection closed")
	if s.conn != nil {
		if err := s.conn.Close(); err != nil {
			LogErrorf("disconnect error: %s", err.Error())
		}
		s.conn = nil
	}
}

func (s *Session) sendPingMessage(message string) error {
	if s.conn == nil {
		return nil
	}
	return s.conn.WriteMessage(ws.PingMessage, []byte(message))
}

func (s *Session) sendPongMessage(message string) error {
	if s.conn == nil {
		return nil
	}
	return s.conn.WriteMessage(ws.PongMessage, []byte(message))
}

func (s *Session) sendTextMessage(message string) error {
	if s.conn == nil {
		return nil
	}
	return s.conn.WriteMessage(ws.TextMessage, []byte(message))
}

func (s *Session) sendBinaryMessage(message []byte) error {
	if s.conn == nil {
		return nil
	}
	return s.conn.WriteMessage(ws.BinaryMessage, message)
}

func (s *Session) writePump() {
	defer s.Close()

	for {
		select {
		case msg := <-s.send:

			s.lastWriteMessageTime = time.Now()

			var err error
			switch s.server.payloadType {
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
	defer s.Close()

	for {
		if s.conn == nil {
			break
		}

		messageType, data, err := s.conn.ReadMessage()
		if err != nil {
			if ws.IsUnexpectedCloseError(err, ws.CloseNormalClosure, ws.CloseGoingAway, ws.CloseAbnormalClosure) {
				LogErrorf("read message error: %v", err)
			}
			return
		}

		s.lastReadMessageTime = time.Now()

		switch messageType {
		case ws.CloseMessage:
			return

		case ws.BinaryMessage:
			_ = s.server.messageHandler(s.SessionID(), data)
			break

		case ws.TextMessage:
			_ = s.server.messageHandler(s.SessionID(), data)
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
