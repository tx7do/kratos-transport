package tcp

import (
	"errors"
	"net"
	"net/url"
	"time"

	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/tx7do/kratos-transport/broker"
)

type ClientMessageHandler func(NetMessagePayload) error

type ClientRawMessageHandler func([]byte) error

type ClientHandlerData struct {
	Handler ClientMessageHandler
	Creator Creator
}
type ClientMessageHandlerMap map[NetMessageType]ClientHandlerData

type Client struct {
	conn net.Conn

	url      string
	endpoint *url.URL

	codec             encoding.Codec
	messageHandlers   ClientMessageHandlerMap
	rawMessageHandler ClientRawMessageHandler

	timeout time.Duration
}

func NewClient(opts ...ClientOption) *Client {
	cli := &Client{
		url:             "",
		timeout:         1 * time.Second,
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

	addr := c.url

	//prefix := "tcp://"
	//if !strings.HasPrefix(addr, "tcp://") {
	//	prefix = "tcp://"
	//}
	//addr = prefix + addr

	c.endpoint, _ = url.Parse(addr)
}

func (c *Client) Connect() error {
	if c.endpoint == nil {
		return errors.New("endpoint is nil")
	}

	LogInfof("connecting to %s", c.endpoint.String())

	conn, err := net.Dial("tcp", c.endpoint.String())
	if err != nil {
		LogErrorf("cant connect to server: %s", err.Error())
		return err
	}

	c.conn = conn

	go c.run()

	return nil
}

func (c *Client) Disconnect() {
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			LogErrorf("disconnect error: %s", err.Error())
		}
		c.conn = nil
	}
}

func (c *Client) RegisterMessageHandler(messageType NetMessageType, handler ClientMessageHandler, binder Creator) {
	if _, ok := c.messageHandlers[messageType]; ok {
		return
	}

	c.messageHandlers[messageType] = ClientHandlerData{handler, binder}
}

func RegisterClientMessageHandler[T any](cli *Client, messageType NetMessageType, handler func(*T) error) {
	cli.RegisterMessageHandler(messageType,
		func(payload NetMessagePayload) error {
			switch t := payload.(type) {
			case *T:
				return handler(t)
			default:
				LogError("invalid payload struct type:", t)
				return errors.New("invalid payload struct type")
			}
		},
		func() any {
			var t T
			return &t
		},
	)
}

func (c *Client) DeregisterMessageHandler(messageType NetMessageType) {
	delete(c.messageHandlers, messageType)
}

func (c *Client) SendRawData(message []byte) error {
	if _, err := c.conn.Write(message); err != nil {
		return err
	}
	return nil
}

func (c *Client) SendMessage(messageType int, message interface{}) error {
	var msg NetPacket
	msg.Type = NetMessageType(messageType)
	msg.Payload, _ = broker.Marshal(c.codec, message)

	var err error

	var buff []byte
	if buff, err = msg.Marshal(); err != nil {
		return err
	}

	return c.SendRawData(buff)
}

func (c *Client) run() {
	defer c.Disconnect()

	buf := make([]byte, 102400)

	var err error
	var readLen int

	for {
		if readLen, err = c.conn.Read(buf); err != nil {
			LogErrorf("read message error: %v", err)
			return
		}

		if c.rawMessageHandler != nil {
			if err := c.rawMessageHandler(buf[:readLen]); err != nil {
				LogErrorf("raw data handler exception: %s", err)
				continue
			}
			continue
		}

		if err = c.messageHandler(buf[:readLen]); err != nil {
			LogErrorf("process message error: %v", err)
		}
	}
}

func (c *Client) messageHandler(buf []byte) error {
	var msg NetPacket
	if err := msg.Unmarshal(buf); err != nil {
		LogErrorf("decode message exception: %s", err)
		return err
	}

	handlerData, ok := c.messageHandlers[msg.Type]
	if !ok {
		LogError("message type not found:", msg.Type)
		return errors.New("message handler not found")
	}

	var payload NetMessagePayload

	if handlerData.Creator != nil {
		payload = handlerData.Creator()
	} else {
		payload = msg.Payload
	}

	if err := broker.Unmarshal(c.codec, msg.Payload, &payload); err != nil {
		LogErrorf("unmarshal message exception: %s", err)
		return err
	}

	if err := handlerData.Handler(payload); err != nil {
		LogErrorf("message handler exception: %s", err)
		return err
	}

	return nil
}
