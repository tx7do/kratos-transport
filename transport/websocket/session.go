package websocket

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	ws "github.com/gorilla/websocket"
)

const channelBufSize = 256

type SessionID string

type Session struct {
	id     SessionID
	conn   *ws.Conn
	send   chan []byte
	server *Server
}

type SessionMap map[SessionID]*Session

func NewSession(conn *ws.Conn, server *Server) *Session {
	if conn == nil {
		panic("conn cannot be nil")
	}

	u1, _ := uuid.NewUUID()

	c := &Session{
		id:     SessionID(u1.String()),
		conn:   conn,
		send:   make(chan []byte, channelBufSize),
		server: server,
	}

	return c
}

func (c *Session) Conn() *ws.Conn {
	return c.conn
}

func (c *Session) SessionID() SessionID {
	return c.id
}

func (c *Session) SendMessage(message []byte) {
	select {
	case c.send <- message:
	}
}

func (c *Session) Close() {
	c.server.unregister <- c
	c.closeConnect()
}

func (c *Session) Listen() {
	go c.writePump()
	go c.readPump()
}

func (c *Session) closeConnect() {
	//log.Info(c.SessionID(), " connection closed")
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			log.Errorf("[websocket] disconnect error: %s", err.Error())
		}
		c.conn = nil
	}
}

func (c *Session) sendPingMessage(message string) error {
	return c.conn.WriteMessage(ws.PingMessage, []byte(message))
}

func (c *Session) sendPongMessage(message string) error {
	return c.conn.WriteMessage(ws.PongMessage, []byte(message))
}

func (c *Session) sendTextMessage(message string) error {
	return c.conn.WriteMessage(ws.TextMessage, []byte(message))
}

func (c *Session) sendBinaryMessage(message []byte) error {
	return c.conn.WriteMessage(ws.BinaryMessage, message)
}

func (c *Session) writePump() {
	defer c.Close()

	for {
		select {
		case msg := <-c.send:
			if err := c.sendBinaryMessage(msg); err != nil {
				log.Error("[websocket] write message error: ", err)
				return
			}
		}
	}
}

func (c *Session) readPump() {
	defer c.Close()

	for {
		messageType, data, err := c.conn.ReadMessage()
		if err != nil {
			if ws.IsUnexpectedCloseError(err, ws.CloseNormalClosure, ws.CloseGoingAway, ws.CloseAbnormalClosure) {
				log.Errorf("[websocket] read message error: %v", err)
			}
			return
		}

		switch messageType {
		case ws.CloseMessage:
			return
		case ws.BinaryMessage:
			_ = c.server.messageHandler(c.SessionID(), data)
			break
		case ws.PingMessage:
			if err := c.sendPongMessage(""); err != nil {
				log.Error("[websocket] write pong message error: ", err)
				return
			}
			break
		case ws.PongMessage:
			break
		case ws.TextMessage:
			log.Error("[websocket] not support text message")
			break
		}

	}
}
