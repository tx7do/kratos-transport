package webtransport

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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

	ctx       context.Context
	ctxCancel context.CancelFunc

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

	if c.transport.QUICConfig == nil {
		c.transport.QUICConfig = &quic.Config{}
	}
	if c.transport.QUICConfig.MaxIncomingStreams == 0 {
		c.transport.QUICConfig.MaxIncomingStreams = 100
	}
}

func (c *Client) Connect() error {
	req, err := c.newWebTransportRequest()
	if err != nil {
		return err
	}

	rsp, err := c.transport.RoundTripOpt(req,
		http3.RoundTripOpt{
			OnlyCachedConn: true,
		},
	)
	if err != nil {
		return err
	}
	if rsp.StatusCode < 200 || rsp.StatusCode >= 300 {
		return fmt.Errorf("received status %d", rsp.StatusCode)
	}

	LogInfof("client connected to: %s", c.url)

	return nil
}

func (c *Client) Disconnect() error {
	LogInfo("client stopping")
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
	return nil
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
		LogErrorf("decode message exception: %s", err)
		return err
	}

	handlerData, ok := c.messageHandlers[msg.Type]
	if !ok {
		LogError("message type not found:", msg.Type)
		return errors.New("message handler not found")
	}

	var payload MessagePayload

	if handlerData.Binder != nil {
		payload = handlerData.Binder()
	} else {
		payload = msg.Body
	}

	if err := broker.Unmarshal(c.codec, msg.Body, &payload); err != nil {
		LogErrorf("unmarshal message exception: %s", err)
		return err
	}

	if err := handlerData.Handler(payload); err != nil {
		LogErrorf("message handler exception: %s", err)
		return err
	}

	return nil
}
