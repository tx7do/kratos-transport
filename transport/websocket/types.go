package websocket

import "net/url"

type Binder func() Any

// ConnectHandler 连接处理器
type ConnectHandler func(sessionId SessionID, queries url.Values, connect bool)

// MessageHandler 消息处理器
type MessageHandler func(SessionID, MessagePayload) error

// MessageHeaderUnmarshaler 解析消息头处理器
type MessageHeaderUnmarshaler func(buf []byte) (messageType MessageType, headerLen int, err error)

type HandlerData struct {
	Handler MessageHandler
	Binder  Binder
}
type MessageHandlerMap map[MessageType]*HandlerData
