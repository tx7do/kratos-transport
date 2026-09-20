package kcp

import (
	"sync"
	"time"

	"github.com/tx7do/go-utils/id"
	"github.com/xtaci/kcp-go/v5"
)

var channelBufSize = 256
var recvBufferSize = 256000

type SessionID string

// SessionHooks defines callbacks used by Session to interact with upper-layer server logic.
type SessionHooks interface {
	removeSession(*Session)
	handleSocketRawData(SessionID, []byte) error
}

type Session struct {
	id     SessionID
	conn   *kcp.UDPSession
	connMu sync.RWMutex
	send   chan []byte
	done   chan struct{}

	// idleTimeout 单次读/写操作的超时；0 表示不设置（默认）
	idleTimeout time.Duration
	listenOnce  sync.Once
	closeOnce   sync.Once
	wg          sync.WaitGroup
	hooks       SessionHooks
}

func NewSession(conn *kcp.UDPSession, hooks SessionHooks) *Session {
	if conn == nil {
		panic("conn cannot be nil")
	}

	c := &Session{
		id:    SessionID(id.NewGUIDv4(false)),
		conn:  conn,
		send:  make(chan []byte, channelBufSize),
		done:  make(chan struct{}),
		hooks: hooks,
	}

	return c
}

func (c *Session) Conn() *kcp.UDPSession {
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
			// 写入带长度前缀的帧，保证对端能正确分包
			if c.idleTimeout > 0 {
				_ = conn.SetWriteDeadline(time.Now().Add(c.idleTimeout))
			}

			if err := WriteFrame(conn, msg); err != nil {
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

		if c.idleTimeout > 0 {
			_ = conn.SetReadDeadline(time.Now().Add(c.idleTimeout))
		}

		// 按帧读取，解决粘包/拆包问题
		frame, err := ReadFrame(conn)
		if err != nil {
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

		if err = c.hooks.handleSocketRawData(c.SessionID(), frame); err != nil {
			LogErrorf("process message error: %v", err)
		}
	}
}
