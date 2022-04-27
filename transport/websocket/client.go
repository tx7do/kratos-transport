package websocket

import (
	"github.com/google/uuid"
	ws "github.com/gorilla/websocket"
	"log"
)

const channelBufSize = 256

type Client struct {
	id     string
	conn   *ws.Conn
	send   chan *Message
	server *Server
}

type ClientMap map[string]*Client
type ClientArray []*Client

func NewClient(conn *ws.Conn, server *Server) *Client {
	if conn == nil {
		panic("conn cannot be nil")
	}

	u1, _ := uuid.NewUUID()

	c := &Client{
		id:     u1.String(),
		conn:   conn,
		send:   make(chan *Message, channelBufSize),
		server: server,
	}

	return c
}

func (c *Client) Conn() *ws.Conn {
	return c.conn
}

func (c *Client) ConnectionID() string {
	return c.id
}

func (c *Client) SendMessage(message *Message) {
	select {
	case c.send <- message:
	}
}

func (c *Client) Close() {
	c.server.unregister <- c
	c.closeConnect()
}

func (c *Client) Listen() {
	go c.writePump()
	go c.readPump()
}

func (c *Client) closeConnect() {
	//log.Println(c.ConnectionID(), " connection closed")
	err := c.conn.Close()
	if err != nil {
		log.Println("close connection error:", err.Error())
	}
	c.conn = nil
}

func (c *Client) writePump() {
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

func (c *Client) readPump() {
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
				replyMsg, _ := c.server.echoHandler(c.ConnectionID(), &Message{Body: data})
				if replyMsg != nil {
					c.SendMessage(replyMsg)
				}
			} else if c.server.readHandler != nil {
				_ = c.server.readHandler(c.ConnectionID(), &Message{Body: data})
			}

		}
	}
}
