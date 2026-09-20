package webrtc

import (
	"net/url"

	"github.com/pion/webrtc/v4"
)

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

// 内置信令消息类型（媒体协商），通过 BinaryNetPacket 承载
const (
	// MsgTypeSignalRenegotiation 重协商信令：
	// 服务端→客户端携带 Offer（新增下行轨道）；客户端→服务端携带 Answer；
	// 客户端→服务端也可携带 Offer（新增上行轨道），服务端会回 Answer
	MsgTypeSignalRenegotiation NetMessageType = 0x10001
	// MsgTypeSignalTrackAvailable 轨道可用广播通知
	MsgTypeSignalTrackAvailable NetMessageType = 0x10002
)

// SignalRenegotiationMsg 重协商信令载荷：Offer/Answer 二选一，由方向决定
type SignalRenegotiationMsg struct {
	Type      string                     `json:"type"` // "renegotiation"
	SessionID SessionID                  `json:"session_id,omitempty"`
	Offer     *webrtc.SessionDescription `json:"offer,omitempty"`
	Answer    *webrtc.SessionDescription `json:"answer,omitempty"`
}

// SignalTrackAvailableMsg 轨道可用通知载荷
type SignalTrackAvailableMsg struct {
	Type        string `json:"type"` // "track_available"
	PublisherID string `json:"publisher_id"`
	TrackID     string `json:"track_id"`
	Kind        string `json:"kind"`
	Codec       string `json:"codec"`
}
