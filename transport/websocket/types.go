package websocket

import "net/url"

// SocketConnectHandler socket connect handler
type SocketConnectHandler func(sessionId SessionID, queries url.Values, connect bool)

// SocketRawDataHandler socket raw data handler
type SocketRawDataHandler func(sessionId SessionID, buf []byte) error

type NetPacketMarshaler func(messageType NetMessageType, message MessagePayload) ([]byte, error)
type NetPacketUnmarshaler func(buf []byte) (*MessageHandlerData, MessagePayload, error)

// NetMessageHandler net message handler
type NetMessageHandler func(SessionID, MessagePayload) error

type Creator func() any

type MessageHandlerData struct {
	Handler NetMessageHandler
	Creator Creator
}

func (h *MessageHandlerData) Create() any {
	if h.Creator != nil {
		return h.Creator()
	}
	return nil
}

type NetMessageHandlerMap map[NetMessageType]*MessageHandlerData
