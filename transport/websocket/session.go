package websocket

import (
	"github.com/google/uuid"
	ws "github.com/gorilla/websocket"
)

var channelBufSize = 256

type SessionID string

type Session struct {
	id     SessionID
	conn   *ws.Conn
	send   chan []byte
	server *Server
}

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
	//LogInfo(c.SessionID(), " connection closed")
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			LogErrorf("disconnect error: %s", err.Error())
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
			var err error
			switch c.server.payloadType {
			case PayloadTypeBinary:
				if err = c.sendBinaryMessage(msg); err != nil {
					LogError("write binary message error: ", err)
					return
				}
				break

			case PayloadTypeText:
				if err = c.sendTextMessage(string(msg)); err != nil {
					LogError("write text message error: ", err)
					return
				}
				break
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
				LogErrorf("read message error: %v", err)
			}
			return
		}

		switch messageType {
		case ws.CloseMessage:
			return

		case ws.BinaryMessage:
			_ = c.server.messageHandler(c.SessionID(), data)
			break

		case ws.TextMessage:
			_ = c.server.messageHandler(c.SessionID(), data)
			break

		case ws.PingMessage:
			if err := c.sendPongMessage(""); err != nil {
				LogError("write pong message error: ", err)
				return
			}
			break

		case ws.PongMessage:
			break
		}

	}
}
