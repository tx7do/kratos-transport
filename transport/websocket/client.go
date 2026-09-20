package websocket

import (
	"encoding/json"
	"errors"
	"net/url"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/encoding"

	ws "github.com/gorilla/websocket"

	"github.com/tx7do/kratos-transport/broker"
)

type ClientMessageHandler func(MessagePayload) error

type ClientHandlerData struct {
	Handler ClientMessageHandler
	Creator Creator
}
type ClientMessageHandlerMap map[NetMessageType]*ClientHandlerData

type Client struct {
	conn *ws.Conn

	connMu    sync.RWMutex
	writeMu   sync.Mutex
	handlerMu sync.RWMutex

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
	c.connMu.Lock()
	c.conn = conn
	c.connMu.Unlock()

	go c.run()

	return nil
}

func (c *Client) Disconnect() {
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

func (c *Client) RegisterMessageHandler(messageType NetMessageType, handler ClientMessageHandler, binder Creator) {
	c.handlerMu.Lock()
	defer c.handlerMu.Unlock()

	if _, ok := c.messageHandlers[messageType]; ok {
		return
	}

	c.messageHandlers[messageType] = &ClientHandlerData{handler, binder}
}

func RegisterClientMessageHandler[T any](cli *Client, messageType NetMessageType, handler func(*T) error) {
	cli.RegisterMessageHandler(messageType,
		func(payload MessagePayload) error {
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
	c.handlerMu.Lock()
	defer c.handlerMu.Unlock()
	delete(c.messageHandlers, messageType)
}

func (c *Client) marshalMessage(messageType NetMessageType, message MessagePayload) ([]byte, error) {
	var err error
	var buff []byte

	switch c.payloadType {
	case PayloadTypeBinary:
		var msg BinaryNetPacket
		msg.Type = messageType
		msg.Payload, err = broker.Marshal(c.codec, message)
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
		var msg TextNetPacket
		msg.Type = messageType
		buf, err = broker.Marshal(c.codec, message)
		msg.Payload = string(buf)
		if err != nil {
			return nil, err
		}
		buff, err = json.Marshal(msg)
		if err != nil {
			return nil, err
		}
		break
	}

	//LogInfo("defaultMarshalNetPacket:", string(buff))

	return buff, nil
}

func (c *Client) SendMessage(messageType NetMessageType, message any) error {
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
	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()
	if conn == nil {
		return errors.New("websocket: client is not connected")
	}
	// gorilla 单写者约束：控制帧必须走 WriteControl
	return conn.WriteControl(ws.PingMessage, []byte(message), time.Now().Add(5*time.Second))
}

func (c *Client) sendPongMessage(message string) error {
	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()
	if conn == nil {
		return errors.New("websocket: client is not connected")
	}
	return conn.WriteControl(ws.PongMessage, []byte(message), time.Now().Add(5*time.Second))
}

func (c *Client) sendTextMessage(message string) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()
	if conn == nil {
		return errors.New("websocket: client is not connected")
	}
	return conn.WriteMessage(ws.TextMessage, []byte(message))
}

func (c *Client) sendBinaryMessage(message []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()
	if conn == nil {
		return errors.New("websocket: client is not connected")
	}
	return conn.WriteMessage(ws.BinaryMessage, message)
}

func (c *Client) run() {
	defer c.Disconnect()

	for {
		c.connMu.RLock()
		conn := c.conn
		c.connMu.RUnlock()
		if conn == nil {
			return
		}

		messageType, data, err := conn.ReadMessage()
		if err != nil {
			if ws.IsUnexpectedCloseError(err, ws.CloseNormalClosure, ws.CloseGoingAway, ws.CloseAbnormalClosure) {
				LogErrorf("read message error: %v", err)
			}
			return
		}

		// gorilla ReadMessage 不返回控制帧（内部处理），
		// 只有 Data 帧会到这里
		switch messageType {
		case ws.BinaryMessage, ws.TextMessage:
			_ = c.messageHandler(data)
		}

	}
}

func (c *Client) unmarshalMessage(buf []byte) (*ClientHandlerData, MessagePayload, error) {
	var handler *ClientHandlerData
	var payload MessagePayload

	switch c.payloadType {
	case PayloadTypeBinary:
		var msg BinaryNetPacket
		if err := msg.Unmarshal(buf); err != nil {
			LogErrorf("decode message exception: %s", err)
			return nil, nil, err
		}

		c.handlerMu.RLock()
		handler, ok := c.messageHandlers[msg.Type]
		c.handlerMu.RUnlock()
		if !ok {
			LogError("message handler not found:", msg.Type)
			return nil, nil, errors.New("message handler not found")
		}

		if handler.Creator != nil {
			payload = handler.Creator()

			if err := broker.Unmarshal(c.codec, msg.Payload, &payload); err != nil {
				LogErrorf("unmarshal message exception: %s", err)
				return nil, nil, err
			}
		} else {
			payload = msg.Payload
		}

		//LogDebug(string(msg.Payload))

	case PayloadTypeText:
		var msg TextNetPacket
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

		if handler.Creator != nil {
			payload = handler.Creator()

			if err := broker.Unmarshal(c.codec, []byte(msg.Payload), &payload); err != nil {
				LogErrorf("unmarshal message exception: %s", err)
				return nil, nil, err
			}
		} else {
			payload = msg.Payload
		}

		//LogDebug(string(msg.Payload))
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
