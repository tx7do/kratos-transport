package tcp

import (
	"net"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

const channelBufSize = 256

type SessionID string

type Session struct {
	id     SessionID
	conn   net.Conn
	send   chan []byte
	server *Server
}

type SessionMap map[SessionID]*Session

func NewSession(conn net.Conn, server *Server) *Session {
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

func (c *Session) Conn() net.Conn {
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
			log.Errorf("[tcp] disconnect error: %s", err.Error())
		}
		c.conn = nil
	}
}

func (c *Session) writePump() {
	defer c.Close()

	for {
		select {
		case msg := <-c.send:
			//var len int
			var err error
			if _, err = c.conn.Write(msg); err != nil {
				log.Error("[tcp] write message error: ", err)
				return
			}
		}
	}
}

func (c *Session) readPump() {
	defer c.Close()

	buf := make([]byte, 102400)
	var err error
	var readLen int

	for {
		if readLen, err = c.conn.Read(buf); err != nil {
			log.Errorf("[tcp] read message error: %v", err)
			return
		}

		if err = c.server.messageHandler(c.SessionID(), buf[:readLen]); err != nil {
			log.Errorf("[tcp] process message error: %v", err)
		}
	}
}
