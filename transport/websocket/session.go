package websocket

import (
	"github.com/google/uuid"
	ws "github.com/gorilla/websocket"
	"log"
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
	//log.Println(c.SessionID(), " connection closed")
	err := c.conn.Close()
	if err != nil {
		log.Println("close connection error:", err.Error())
	}
	c.conn = nil
}

func (c *Session) writePump() {
	defer c.Close()

	for {
		select {
		case msg := <-c.send:
			if err := c.conn.WriteMessage(ws.BinaryMessage, msg); err != nil {
				log.Println("write message error: ", err)
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
				log.Printf("read message error: %v", err)
			}
			return
		} else if messageType != ws.BinaryMessage {
			log.Println("Non binary message received, ignoring")
		} else {
			_ = c.server.messageHandler(c.SessionID(), data)
		}
	}
}
