package websocket

import (
	"encoding/json"
	"errors"
	"net/url"
	"time"

	"github.com/go-kratos/kratos/v2/encoding"
	ws "github.com/gorilla/websocket"
	"github.com/tx7do/kratos-transport/broker"
)

type ClientMessageHandler func(MessagePayload) error

type ClientHandlerData struct {
	Handler ClientMessageHandler
	Binder  Binder
}
type ClientMessageHandlerMap map[MessageType]*ClientHandlerData

type Client struct {
	conn *ws.Conn

	url      string
	endpoint *url.URL

	codec           encoding.Codec
	messageHandlers ClientMessageHandlerMap

	timeout time.Duration

	payloadType PayloadType
}

func NewClient(opts ...ClientOption) *Client {
	cli := &Client{
		url:             "",
		timeout:         1 * time.Second,
		codec:           encoding.GetCodec("json"),
		messageHandlers: make(ClientMessageHandlerMap),
		payloadType:     PayloadTypeBinary,
	}

	cli.init(opts...)

	return cli
}

func (c *Client) init(opts ...ClientOption) {
	for _, o := range opts {
		o(c)
	}

	c.endpoint, _ = url.Parse(c.url)
}

func (c *Client) Connect() error {
	if c.endpoint == nil {
		return errors.New("endpoint is nil")
	}

	LogInfof("connecting to %s", c.endpoint.String())

	conn, resp, err := ws.DefaultDialer.Dial(c.endpoint.String(), nil)
	if err != nil {
		LogErrorf("%s [%v]", err.Error(), resp)
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

func (c *Client) RegisterMessageHandler(messageType MessageType, handler ClientMessageHandler, binder Binder) {
	if _, ok := c.messageHandlers[messageType]; ok {
		return
	}

	c.messageHandlers[messageType] = &ClientHandlerData{handler, binder}
}

func (c *Client) DeregisterMessageHandler(messageType MessageType) {
	delete(c.messageHandlers, messageType)
}

func (c *Client) marshalMessage(messageType MessageType, message MessagePayload) ([]byte, error) {
	var err error
	var buff []byte

	switch c.payloadType {
	case PayloadTypeBinary:
		var msg BinaryMessage
		msg.Type = messageType
		msg.Body, err = broker.Marshal(c.codec, message)
		if err != nil {
			return nil, err
		}
		buff, err = msg.Marshal()
		if err != nil {
			return nil, err
		}
		break

	case PayloadTypeText:
		var buf []byte
		var msg TextMessage
		msg.Type = messageType
		buf, err = broker.Marshal(c.codec, message)
		msg.Body = string(buf)
		if err != nil {
			return nil, err
		}
		buff, err = json.Marshal(msg)
		if err != nil {
			return nil, err
		}
		break
	}

	//LogInfo("marshalMessage:", string(buff))

	return buff, nil
}

func (c *Client) SendMessage(messageType MessageType, message interface{}) error {
	buff, err := c.marshalMessage(messageType, message)
	if err != nil {
		LogError("marshal message exception:", err)
		return err
	}

	switch c.payloadType {
	case PayloadTypeBinary:
		if err = c.sendBinaryMessage(buff); err != nil {
			return err
		}
		break

	case PayloadTypeText:
		if err = c.sendTextMessage(string(buff)); err != nil {
			return err
		}
		break
	}

	return nil
}

func (c *Client) sendPingMessage(message string) error {
	return c.conn.WriteMessage(ws.PingMessage, []byte(message))
}

func (c *Client) sendPongMessage(message string) error {
	return c.conn.WriteMessage(ws.PongMessage, []byte(message))
}

func (c *Client) sendTextMessage(message string) error {
	return c.conn.WriteMessage(ws.TextMessage, []byte(message))
}

func (c *Client) sendBinaryMessage(message []byte) error {
	return c.conn.WriteMessage(ws.BinaryMessage, message)
}

func (c *Client) run() {
	defer c.Disconnect()

	for {
		messageType, data, err := c.conn.ReadMessage()
		if err != nil {
			if ws.IsUnexpectedCloseError(err, ws.CloseNormalClosure, ws.CloseGoingAway, ws.CloseAbnormalClosure) {
				LogErrorf("read message error: %v", err)
			}
			return
		}

		switch messageType {
		case ws.CloseMessage:
			return

		case ws.BinaryMessage:
			_ = c.messageHandler(data)
			break

		case ws.TextMessage:
			_ = c.messageHandler(data)
			break

		case ws.PingMessage:
			if err := c.sendPongMessage(""); err != nil {
				LogError("write pong message error: ", err)
				return
			}
			break

		case ws.PongMessage:
			break
		}

	}
}

func (c *Client) unmarshalMessage(buf []byte) (*ClientHandlerData, MessagePayload, error) {
	var handler *ClientHandlerData
	var payload MessagePayload

	switch c.payloadType {
	case PayloadTypeBinary:
		var msg BinaryMessage
		if err := msg.Unmarshal(buf); err != nil {
			LogErrorf("decode message exception: %s", err)
			return nil, nil, err
		}

		var ok bool
		handler, ok = c.messageHandlers[msg.Type]
		if !ok {
			LogError("message handler not found:", msg.Type)
			return nil, nil, errors.New("message handler not found")
		}

		if handler.Binder != nil {
			payload = handler.Binder()
		} else {
			payload = msg.Body
		}

		if err := broker.Unmarshal(c.codec, msg.Body, &payload); err != nil {
			LogErrorf("unmarshal message exception: %s", err)
			return nil, nil, err
		}
		//LogDebug(string(msg.Body))

	case PayloadTypeText:
		var msg TextMessage
		if err := msg.Unmarshal(buf); err != nil {
			LogErrorf("decode message exception: %s", err)
			return nil, nil, err
		}

		var ok bool
		handler, ok = c.messageHandlers[msg.Type]
		if !ok {
			LogError("message handler not found:", msg.Type)
			return nil, nil, errors.New("message handler not found")
		}

		if handler.Binder != nil {
			payload = handler.Binder()
		} else {
			payload = msg.Body
		}

		if err := broker.Unmarshal(c.codec, []byte(msg.Body), &payload); err != nil {
			LogErrorf("unmarshal message exception: %s", err)
			return nil, nil, err
		}
		//LogDebug(string(msg.Body))
	}

	return handler, payload, nil
}

func (c *Client) messageHandler(buf []byte) error {
	var err error
	var handler *ClientHandlerData
	var payload MessagePayload

	if handler, payload, err = c.unmarshalMessage(buf); err != nil {
		LogErrorf("unmarshal message failed: %s", err)
		return err
	}
	//LogDebug(payload)

	if err = handler.Handler(payload); err != nil {
		LogErrorf("message handler exception: %s", err)
		return err
	}

	return nil
}
