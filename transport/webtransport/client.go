package webtransport

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/tx7do/kratos-transport/broker"
)

type ClientMessageHandler func(MessagePayload) error

type ClientHandlerData struct {
	Handler ClientMessageHandler
	Binder  Binder
}
type ClientMessageHandlerMap map[MessageType]ClientHandlerData

type Client struct {
	transport *http3.Transport

	connMu  sync.RWMutex
	conn    *quic.Conn
	h3conn  *http3.ClientConn
	stream  *http3.RequestStream
	writeMu sync.Mutex

	ctx       context.Context
	ctxCancel context.CancelFunc

	handlersMu sync.RWMutex
	running    atomic.Bool

	timeout time.Duration
	tlsConf *tls.Config

	url string

	codec           encoding.Codec
	messageHandlers ClientMessageHandlerMap
}

func NewClient(opts ...ClientOption) *Client {
	cli := &Client{
		transport:       &http3.Transport{},
		codec:           encoding.GetCodec("json"),
		messageHandlers: make(ClientMessageHandlerMap),
	}
	cli.init(opts...)
	return cli
}

func (c *Client) init(opts ...ClientOption) {
	for _, o := range opts {
		o(c)
	}

	c.ctx, c.ctxCancel = context.WithCancel(context.Background())

	timeout := c.timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	c.timeout = timeout

	if c.tlsConf == nil {
		c.tlsConf = &tls.Config{
			InsecureSkipVerify: true,
		}
	}
	// QUIC 强制要求 ALPN
	if len(c.tlsConf.NextProtos) == 0 {
		c.tlsConf.NextProtos = []string{alpnQuicTransport}
	}
	c.transport.TLSClientConfig = c.tlsConf

	c.transport.EnableDatagrams = true

	if c.transport.AdditionalSettings == nil {
		c.transport.AdditionalSettings = make(map[uint64]uint64)
	}
	c.transport.AdditionalSettings[settingsEnableWebtransport] = 1

	if c.transport.QUICConfig == nil {
		c.transport.QUICConfig = &quic.Config{}
	}
	if c.transport.QUICConfig.MaxIncomingStreams == 0 {
		c.transport.QUICConfig.MaxIncomingStreams = 100
	}
}

// Connect 建立 QUIC/HTTP3 连接，发送 WebTransport CONNECT 请求并升级为双向消息流。
func (c *Client) Connect() error {
	c.connMu.RLock()
	connected := c.stream != nil
	c.connMu.RUnlock()
	if connected {
		return nil
	}

	// 每次 Connect 使用全新的 ctx：Disconnect 会取消旧 ctx，
	// 不重建的话重连时 DialAddr 会立即 context canceled
	c.ctx, c.ctxCancel = context.WithCancel(context.Background())

	u, err := url.Parse(c.url)
	if err != nil {
		return err
	}

	addr := u.Host
	if u.Port() == "" {
		addr += ":443"
	}

	tlsConf := c.tlsConf.Clone()

	quicConfig := c.transport.QUICConfig
	if quicConfig == nil {
		quicConfig = &quic.Config{}
	}

	conn, err := quic.DialAddr(c.ctx, addr, tlsConf, quicConfig)
	if err != nil {
		return fmt.Errorf("dial quic server %s failed: %w", addr, err)
	}
	// 握手半程失败时回收 QUIC 连接，避免悬挂到 idle 超时
	defer func() {
		if conn != nil {
			_ = conn.CloseWithError(0, "connect failed")
			conn = nil
		}
	}()

	h3conn := c.transport.NewClientConn(conn)

	// 等待服务端 SETTINGS，确认 WebTransport 支持
	select {
	case <-h3conn.ReceivedSettings():
	case <-time.After(c.timeout):
		return errors.New("timeout waiting for server settings")
	}

	str, err := h3conn.OpenRequestStream(c.ctx)
	if err != nil {
		return fmt.Errorf("open request stream failed: %w", err)
	}

	req, err := c.newWebTransportRequest(u)
	if err != nil {
		return err
	}

	if err := str.SendRequestHeader(req); err != nil {
		return fmt.Errorf("send connect request failed: %w", err)
	}

	rsp, err := str.ReadResponse()
	if err != nil {
		return fmt.Errorf("read connect response failed: %w", err)
	}
	if rsp.StatusCode < 200 || rsp.StatusCode >= 300 {
		return fmt.Errorf("received status %d", rsp.StatusCode)
	}

	c.connMu.Lock()
	c.conn = conn
	c.h3conn = h3conn
	c.stream = str
	c.connMu.Unlock()
	conn = nil // 已接管，握手失败的回收 defer 不应触发

	go c.run()

	LogInfof("client connected to: %s", c.url)

	return nil
}

func (c *Client) Disconnect() error {
	LogInfo("client stopping")

	c.connMu.Lock()
	stream := c.stream
	conn := c.conn
	c.stream = nil
	c.conn = nil
	c.h3conn = nil
	c.connMu.Unlock()

	if stream != nil {
		stream.CancelRead(0)
		_ = stream.Close()
	}
	if conn != nil {
		_ = conn.CloseWithError(0, "client closed")
	}

	c.running.Store(false)
	c.ctxCancel()

	return nil
}

func (c *Client) RegisterMessageHandler(messageType MessageType, handler ClientMessageHandler, binder Binder) {
	c.handlersMu.Lock()
	defer c.handlersMu.Unlock()

	if _, ok := c.messageHandlers[messageType]; ok {
		return
	}

	c.messageHandlers[messageType] = ClientHandlerData{handler, binder}
}

func (c *Client) DeregisterMessageHandler(messageType MessageType) {
	c.handlersMu.Lock()
	defer c.handlersMu.Unlock()

	delete(c.messageHandlers, messageType)
}

func (c *Client) SendMessage(messageType int, message any) error {
	var msg Message
	msg.Type = MessageType(messageType)

	body, err := broker.Marshal(c.codec, message)
	if err != nil {
		return err
	}
	msg.Body = body

	buff, err := msg.Marshal()
	if err != nil {
		return err
	}

	return c.SendRawData(buff)
}

func (c *Client) SendRawData(data []byte) error {
	c.connMu.RLock()
	stream := c.stream
	c.connMu.RUnlock()

	if stream == nil {
		return errors.New("client is not connected")
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	return WriteFrame(stream, data)
}

// run 在后台持续读取服务端下行的消息帧并分发。
func (c *Client) run() {
	for {
		c.connMu.RLock()
		stream := c.stream
		c.connMu.RUnlock()

		if stream == nil {
			return
		}

		frame, err := ReadFrame(stream)
		if err != nil {
			select {
			case <-c.ctx.Done():
				return
			default:
			}
			LogErrorf("read message error: %v", err)
			_ = c.Disconnect()
			return
		}

		if err = c.messageHandler(frame); err != nil {
			LogErrorf("process message error: %v", err)
		}
	}
}

func (c *Client) newWebTransportRequest(u *url.URL) (*http.Request, error) {
	hdr := make(http.Header)
	hdr.Add(webTransportDraftOfferHeaderKey, "1")

	req := &http.Request{
		Method: http.MethodConnect,
		Header: hdr,
		Proto:  protocolHeader,
		Host:   u.Host,
		URL:    u,
	}
	req = req.WithContext(c.ctx)

	return req, nil
}

func (c *Client) messageHandler(buf []byte) error {
	var msg Message
	if err := msg.Unmarshal(buf); err != nil {
		LogErrorf("decode message exception: %s", err)
		return err
	}

	c.handlersMu.RLock()
	handlerData, ok := c.messageHandlers[msg.Type]
	c.handlersMu.RUnlock()
	if !ok {
		LogError("message type not found:", msg.Type)
		return errors.New("message handler not found")
	}

	var payload MessagePayload

	if handlerData.Binder != nil {
		payload = handlerData.Binder()

		if err := broker.Unmarshal(c.codec, msg.Body, &payload); err != nil {
			LogErrorf("unmarshal message exception: %s", err)
			return err
		}
	} else {
		payload = msg.Body
	}

	if err := handlerData.Handler(payload); err != nil {
		LogErrorf("message handler exception: %s", err)
		return err
	}

	return nil
}
