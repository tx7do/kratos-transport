package webtransport

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/http3"
	"github.com/lucas-clemente/quic-go/quicvarint"
	"github.com/tx7do/kratos-transport/broker"
	"io"
	"net/http"
	"net/url"
	"time"
)

type ClientMessageHandler func(MessagePayload) error

type ClientHandlerData struct {
	Handler ClientMessageHandler
	Binder  Binder
}
type ClientMessageHandlerMap map[MessageType]ClientHandlerData

type Client struct {
	transport *http3.RoundTripper

	ctx       context.Context
	ctxCancel context.CancelFunc

	timeout time.Duration
	tlsConf *tls.Config

	url string

	sessions *sessionManager
	session  *Session

	codec           encoding.Codec
	messageHandlers ClientMessageHandlerMap
}

func NewClient(opts ...ClientOption) *Client {
	cli := &Client{
		transport:       &http3.RoundTripper{},
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
	c.sessions = newSessionManager(timeout)

	if c.tlsConf == nil {
		c.tlsConf = &tls.Config{
			RootCAs:            generateCertPool(),
			InsecureSkipVerify: true,
		}
	}
	c.transport.TLSClientConfig = c.tlsConf

	c.transport.EnableDatagrams = true

	if c.transport.AdditionalSettings == nil {
		c.transport.AdditionalSettings = make(map[uint64]uint64)
	}

	c.transport.StreamHijacker = func(ft http3.FrameType, conn quic.Connection, str quic.Stream, e error) (hijacked bool, err error) {
		if isWebTransportError(e) {
			return true, nil
		}
		if ft != webTransportFrameType {
			return false, nil
		}
		id, err := quicvarint.Read(quicvarint.NewReader(str))
		if err != nil {
			if isWebTransportError(err) {
				return true, nil
			}
			return false, err
		}
		c.sessions.AddStream(conn, str, SessionID(id))
		return true, nil
	}
	c.transport.UniStreamHijacker = func(st http3.StreamType, conn quic.Connection, str quic.ReceiveStream, err error) (hijacked bool) {
		if st != webTransportUniStreamType && !isWebTransportError(err) {
			return false
		}
		c.sessions.AddUniStream(conn, str)
		return true
	}
	if c.transport.QuicConfig == nil {
		c.transport.QuicConfig = &quic.Config{}
	}
	if c.transport.QuicConfig.MaxIncomingStreams == 0 {
		c.transport.QuicConfig.MaxIncomingStreams = 100
	}
}

func (c *Client) Connect() error {
	req, err := c.newWebTransportRequest()
	if err != nil {
		return err
	}

	rsp, err := c.transport.RoundTripOpt(req,
		http3.RoundTripOpt{
			DontCloseRequestStream: true,
		},
	)
	if err != nil {
		return err
	}
	if rsp.StatusCode < 200 || rsp.StatusCode >= 300 {
		return fmt.Errorf("received status %d", rsp.StatusCode)
	}

	stream := rsp.Body.(http3.HTTPStreamer).HTTPStream()
	session := c.sessions.AddSession(
		rsp.Body.(http3.Hijacker).StreamCreator(),
		SessionID(stream.StreamID()),
		stream,
	)

	c.session = session

	go c.doAcceptStream(session)

	log.Infof("[webtransport] client connected to: %s", c.url)

	return nil
}

func (c *Client) Disconnect() error {
	log.Info("[webtransport] client stopping")
	return nil
}

func (c *Client) RegisterMessageHandler(messageType MessageType, handler ClientMessageHandler, binder Binder) {
	if _, ok := c.messageHandlers[messageType]; ok {
		return
	}

	c.messageHandlers[messageType] = ClientHandlerData{handler, binder}
}

func (c *Client) DeregisterMessageHandler(messageType MessageType) {
	delete(c.messageHandlers, messageType)
}

func (c *Client) SendMessage(messageType int, message interface{}) error {
	var msg Message
	msg.Type = MessageType(messageType)
	msg.Body, _ = broker.Marshal(c.codec, message)

	buff, err := msg.Marshal()
	if err != nil {
		return err
	}

	if err := c.SendRawData(buff); err != nil {
		return err
	}

	return nil
}

func (c *Client) SendRawData(data []byte) error {
	if c.session == nil {
		return errors.New("[webtransport] send data failed, not connected")
	}

	stream, err := c.session.OpenStream()
	if err != nil {
		log.Error("[webtransport] open qStream failed: ", err.Error())
		return err
	}
	defer stream.Close()

	_, err = stream.Write(data)
	if err != nil {
		log.Error("[webtransport] write qStream failed: ", err.Error())
		return err
	}

	return nil
}

func (c *Client) doAcceptStream(session *Session) {
	for {
		acceptStream, err := session.AcceptStream(c.ctx)
		if err != nil {
			log.Debug("[webtransport] accept stream failed: ", err.Error())
			break
		}
		data, err := io.ReadAll(acceptStream)
		if err != nil {
			log.Error("[webtransport] read data failed: ", err.Error())
		}
		log.Debug("[webtransport] receive data: ", string(data))
		_ = c.messageHandler(data)
	}
}

func (c *Client) doAcceptUniStream(session *Session) {
	for {
		acceptStream, err := session.AcceptUniStream(c.ctx)
		if err != nil {
			log.Debug("[webtransport] accept stream failed: ", err.Error())
			break
		}
		data, err := io.ReadAll(acceptStream)
		if err != nil {
			log.Error("[webtransport] read data failed: ", err.Error())
		}
		log.Debug("[webtransport] receive data: ", string(data))
		_ = c.messageHandler(data)
	}
}

func (c *Client) newWebTransportRequest() (*http.Request, error) {
	u, err := url.Parse(c.url)
	if err != nil {
		return nil, err
	}

	hdr := make(http.Header)
	hdr.Add(webTransportDraftOfferHeaderKey, "1")

	req := &http.Request{
		Method: http.MethodConnect,
		Header: hdr,
		Proto:  "webtransport",
		Host:   u.Host,
		URL:    u,
	}
	req = req.WithContext(c.ctx)

	return req, err
}

func (c *Client) messageHandler(buf []byte) error {
	var msg Message
	if err := msg.Unmarshal(buf); err != nil {
		log.Errorf("[webtransport] decode message exception: %s", err)
		return err
	}

	handlerData, ok := c.messageHandlers[msg.Type]
	if !ok {
		log.Error("[webtransport] message type not found:", msg.Type)
		return errors.New("message handler not found")
	}

	var payload MessagePayload

	if handlerData.Binder != nil {
		payload = handlerData.Binder()
	}

	if err := broker.Unmarshal(c.codec, msg.Body, payload); err != nil {
		log.Errorf("[webtransport] unmarshal message exception: %s", err)
		return err
	}

	if err := handlerData.Handler(payload); err != nil {
		log.Errorf("[webtransport] message handler exception: %s", err)
		return err
	}

	return nil
}
