package websocket

import (
	ws "github.com/gorilla/websocket"
	"log"
)

const channelBufSize = 100

type Client struct {
	conn   *ws.Conn
	ch     chan *Message
	doneCh chan bool
	server *Server
}

type ClientArray []*Client

func NewClient(conn *ws.Conn, server *Server) *Client {
	if conn == nil {
		panic("conn cannot be nil")
	}

	ch := make(chan *Message, channelBufSize)
	doneCh := make(chan bool)

	return &Client{conn, ch, doneCh, server}
}

func (c *Client) Conn() *ws.Conn {
	return c.conn
}

func (c *Client) SendMessage(message *Message) {
	select {
	case c.ch <- message:
	}
}

func (c *Client) Done() {
	c.doneCh <- true
}

func (c *Client) Listen() {
	go c.writePump()
	c.readPump()
}

func (c *Client) closeConnect() {
	err := c.conn.Close()
	if err != nil {
		log.Println("Error:", err.Error())
	}
	c.conn = nil
}

func (c *Client) writePump() {
	defer c.closeConnect()

	for {
		select {

		case msg := <-c.ch:
			if err := c.conn.WriteMessage(ws.BinaryMessage, msg.Body); err != nil {
				return
			}

		case <-c.doneCh:
			c.doneCh <- true
			return
		}
	}
}

func (c *Client) readPump() {
	defer c.closeConnect()

	for {
		select {

		case <-c.doneCh:
			c.doneCh <- true
			return

		default:
			c.readFromWebSocket()
		}
	}
}

func (c *Client) readFromWebSocket() {
	messageType, data, err := c.conn.ReadMessage()
	if err != nil {
		log.Println(err)
		c.doneCh <- true
	} else if messageType != ws.BinaryMessage {
		log.Println("Non binary message received, ignoring")
	} else {
		replyMsg, _ := c.server.handler(&Message{Body: data})
		if replyMsg != nil {
			c.SendMessage(replyMsg)
		}
	}
}
