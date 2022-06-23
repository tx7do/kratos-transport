package websocket

import (
	"github.com/google/uuid"
	ws "github.com/gorilla/websocket"
	"log"
)

const channelBufSize = 256

type Session struct {
	id     string
	conn   *ws.Conn
	send   chan *Message
	server *Server
}

type SessionMap map[string]*Session
type SessionArray []*Session

func NewSession(conn *ws.Conn, server *Server) *Session {
	if conn == nil {
		panic("conn cannot be nil")
	}

	u1, _ := uuid.NewUUID()

	c := &Session{
		id:     u1.String(),
		conn:   conn,
		send:   make(chan *Message, channelBufSize),
		server: server,
	}

	return c
}

func (c *Session) Conn() *ws.Conn {
	return c.conn
}

func (c *Session) SessionID() string {
	return c.id
}

func (c *Session) SendMessage(message *Message) {
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
			if err := c.conn.WriteMessage(ws.BinaryMessage, msg.Body); err != nil {
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
			if c.server.echoHandler != nil {
				replyMsg, _ := c.server.echoHandler(c.SessionID(), &Message{Body: data})
				if replyMsg != nil {
					c.SendMessage(replyMsg)
				}
			} else if c.server.readHandler != nil {
				_ = c.server.readHandler(c.SessionID(), &Message{Body: data})
			}

		}
	}
}
