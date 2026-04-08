package tcp

import (
	"net"
	"sync"

	"github.com/tx7do/go-utils/id"
)

var channelBufSize = 256
var recvBufferSize = 256000

type SessionID string

type Session struct {
	id SessionID

	conn  net.Conn
	hooks SessionHooks

	send chan []byte

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
}

func NewSession(conn net.Conn, hooks SessionHooks) *Session {
	if conn == nil {
		panic("conn cannot be nil")
	}

	c := &Session{
		id:    SessionID(id.NewGUIDv4(false)),
		conn:  conn,
		done:  make(chan struct{}),
		send:  make(chan []byte, channelBufSize),
		hooks: hooks,
	}

	return c
}

func (c *Session) Conn() net.Conn {
	c.connMu.RLock()
	defer c.connMu.RUnlock()

	return c.conn
}

func (c *Session) SessionID() SessionID {
	return c.id
}

func (c *Session) SendMessage(message []byte) {
	select {
	case <-c.done:
		return
	case c.send <- message:
	}
}

func (c *Session) Close() {
	c.closeOnce.Do(func() {
		close(c.done)
		c.closeConnect()

		if c.hooks != nil {
			c.hooks.removeSession(c)
		}
	})
}

func (c *Session) Listen() {
	c.listenOnce.Do(func() {
		c.wg.Add(2)

		go c.writePump()
		go c.readPump()
	})
}

func (c *Session) Wait() {
	c.wg.Wait()
}

func (c *Session) closeConnect() {
	//LogInfo(c.SessionID(), " connection closed")
	c.connMu.Lock()
	conn := c.conn
	c.conn = nil
	c.connMu.Unlock()

	if conn != nil {
		if err := conn.Close(); err != nil {
			LogErrorf("disconnect error: %s", err.Error())
		}
	}
}

func (c *Session) writePump() {
	defer c.wg.Done()
	defer c.Close()

	for {
		select {
		case <-c.done:
			return

		case msg := <-c.send:
			conn := c.Conn()
			if conn == nil {
				return
			}
			var err error
			if _, err = conn.Write(msg); err != nil {
				select {
				case <-c.done:
					return
				default:
				}
				LogError("write message error: ", err)
				return
			}
		}
	}
}

func (c *Session) readPump() {
	defer c.wg.Done()
	defer c.Close()

	buf := make([]byte, recvBufferSize)
	var err error
	var readLen int

	for {
		select {
		case <-c.done:
			return
		default:
		}

		conn := c.Conn()
		if conn == nil {
			return
		}

		if readLen, err = conn.Read(buf); err != nil {
			select {
			case <-c.done:
				return
			default:
			}
			LogErrorf("read message error: %v", err)
			return
		}

		if c.hooks == nil {
			continue
		}

		if err = c.hooks.handleSocketRawData(c.SessionID(), buf[:readLen]); err != nil {
			LogErrorf("process message error: %v", err)
		}
	}
}
