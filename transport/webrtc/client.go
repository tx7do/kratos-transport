package webrtc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/pion/webrtc/v4"

	"github.com/tx7do/kratos-transport/broker"
)

type ClientMessageHandler func(MessagePayload) error

type ClientHandlerData struct {
	Handler ClientMessageHandler
	Creator Creator
}

type ClientMessageHandlerMap map[NetMessageType]*ClientHandlerData

type Client struct {
	signalURL     string
	authorization string

	codec       encoding.Codec
	payloadType PayloadType

	dataChannelLabel string
	webrtcConfig     webrtc.Configuration
	connectTimeout   time.Duration
	signalTimeout    time.Duration

	pcMu sync.RWMutex
	pc   *webrtc.PeerConnection
	dc   *webrtc.DataChannel

	handlerMu       sync.RWMutex
	messageHandlers ClientMessageHandlerMap

	// 媒体轨道相关
	localTracks  map[string]*webrtc.TrackLocalStaticRTP
	remoteTracks map[string]*webrtc.TrackRemote
}

func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		codec:            encoding.GetCodec("json"),
		payloadType:      PayloadTypeBinary,
		dataChannelLabel: "kratos",
		webrtcConfig:     webrtc.Configuration{},
		connectTimeout:   10 * time.Second,
		signalTimeout:    10 * time.Second,
		messageHandlers:  make(ClientMessageHandlerMap),
		localTracks:      make(map[string]*webrtc.TrackLocalStaticRTP),
		remoteTracks:     make(map[string]*webrtc.TrackRemote),
	}

	// 内置信令处理器：处理服务端下发的重协商 Offer（服务端新增下行轨道时）。
	// 在用户 opts 之前注册：RegisterMessageHandler 先注册者优先，用户无法覆盖
	c.RegisterMessageHandler(MsgTypeSignalRenegotiation,
		func(payload MessagePayload) error {
			return c.handleSignalRenegotiation(payload)
		},
		func() any {
			return &SignalRenegotiationMsg{}
		},
	)

	for _, opt := range opts {
		opt(c)
	}

	if c.codec == nil {
		c.codec = encoding.GetCodec("json")
	}

	return c
}

func (c *Client) RegisterMessageHandler(messageType NetMessageType, handler ClientMessageHandler, binder Creator) {
	c.handlerMu.Lock()
	defer c.handlerMu.Unlock()

	if _, ok := c.messageHandlers[messageType]; ok {
		return
	}

	c.messageHandlers[messageType] = &ClientHandlerData{Handler: handler, Creator: binder}
}

func RegisterClientMessageHandler[T any](cli *Client, messageType NetMessageType, handler func(*T) error) {
	cli.RegisterMessageHandler(messageType,
		func(payload MessagePayload) error {
			switch t := payload.(type) {
			case *T:
				return handler(t)
			default:
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

func (c *Client) Connect() error {
	if c.signalURL == "" {
		return errors.New("signal url is empty")
	}

	pc, err := webrtc.NewPeerConnection(c.webrtcConfig)
	if err != nil {
		return err
	}

	openCh := make(chan struct{})
	openOnce := sync.Once{}

	dc, err := pc.CreateDataChannel(c.dataChannelLabel, nil)
	if err != nil {
		_ = pc.Close()
		return err
	}

	dc.OnOpen(func() {
		openOnce.Do(func() { close(openCh) })
	})
	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		if err := c.messageHandler(msg.Data); err != nil {
			LogErrorf("client message handler failed: %s", err)
		}
	})
	dc.OnError(func(err error) {
		if err != nil && !isExpectedDataChannelCloseError(err) {
			LogErrorf("client data channel error: %s", err)
		}
	})

	// 处理 incoming 媒体轨道
	pc.OnTrack(func(remote *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		LogInfof("received remote track: kind=%s, codec=%s", remote.Kind(), remote.Codec().MimeType)

		c.pcMu.Lock()
		c.remoteTracks[remote.ID()] = remote
		c.pcMu.Unlock()

		// 这里可以添加回调通知应用层有新的媒体轨道
		// 例如: c.onRemoteTrack(remote, receiver)
	})

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		switch state {
		case webrtc.PeerConnectionStateClosed, webrtc.PeerConnectionStateFailed:
			_ = c.Disconnect()
		case webrtc.PeerConnectionStateDisconnected:
			// 瞬时态：ICE 可能自愈恢复 Connected，不做永久断开
			LogWarn("peer connection disconnected, waiting for ICE recovery")
		}
	})

	offer, err := pc.CreateOffer(nil)
	if err != nil {
		_ = pc.Close()
		return err
	}

	gatherDone := webrtc.GatheringCompletePromise(pc)
	if err = pc.SetLocalDescription(offer); err != nil {
		_ = pc.Close()
		return err
	}
	<-gatherDone

	local := pc.LocalDescription()
	if local == nil {
		_ = pc.Close()
		return errors.New("local description is nil")
	}

	answer, err := c.sendOffer(*local)
	if err != nil {
		_ = pc.Close()
		return err
	}

	if err = pc.SetRemoteDescription(answer); err != nil {
		_ = pc.Close()
		return err
	}

	c.pcMu.Lock()
	c.pc = pc
	c.dc = dc
	c.pcMu.Unlock()

	select {
	case <-openCh:
		return nil
	case <-time.After(c.connectTimeout):
		_ = c.Disconnect()
		return errors.New("wait data channel open timeout")
	}
}

func (c *Client) sendOffer(offer webrtc.SessionDescription) (webrtc.SessionDescription, error) {
	payload := map[string]webrtc.SessionDescription{"offer": offer}
	body, err := json.Marshal(payload)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.signalTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.signalURL, bytes.NewReader(body))
	if err != nil {
		return webrtc.SessionDescription{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.authorization != "" {
		req.Header.Set("Authorization", c.authorization)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return webrtc.SessionDescription{}, fmt.Errorf("signal request failed: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var wrapped signalResponse
	if err = json.Unmarshal(respBody, &wrapped); err == nil && wrapped.Answer.Type != 0 {
		return wrapped.Answer, nil
	}

	var answer webrtc.SessionDescription
	if err = json.Unmarshal(respBody, &answer); err != nil {
		return webrtc.SessionDescription{}, err
	}
	if answer.Type == 0 {
		return webrtc.SessionDescription{}, errors.New("invalid answer from signaling server")
	}
	return answer, nil
}

func (c *Client) Disconnect() error {
	c.pcMu.Lock()
	dc := c.dc
	pc := c.pc
	c.dc = nil
	c.pc = nil
	c.pcMu.Unlock()

	var firstErr error
	if dc != nil {
		if err := dc.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if pc != nil {
		if err := pc.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (c *Client) SendMessage(messageType NetMessageType, message any) error {
	buf, err := c.marshalMessage(messageType, message)
	if err != nil {
		return err
	}

	c.pcMu.RLock()
	dc := c.dc
	c.pcMu.RUnlock()
	if dc == nil {
		return errors.New("data channel is nil")
	}

	switch c.payloadType {
	case PayloadTypeText:
		return dc.SendText(string(buf))
	default:
		return dc.Send(buf)
	}
}

func (c *Client) marshalMessage(messageType NetMessageType, message MessagePayload) ([]byte, error) {
	switch c.payloadType {
	case PayloadTypeBinary:
		var msg BinaryNetPacket
		msg.Type = messageType
		payload, err := broker.Marshal(c.codec, message)
		if err != nil {
			return nil, err
		}
		msg.Payload = payload
		return msg.Marshal()
	case PayloadTypeText:
		var msg TextNetPacket
		msg.Type = messageType
		payload, err := broker.Marshal(c.codec, message)
		if err != nil {
			return nil, err
		}
		msg.Payload = string(payload)
		return msg.Marshal()
	default:
		return nil, fmt.Errorf("unsupported payload type: %d", c.payloadType)
	}
}

func (c *Client) unmarshalMessage(buf []byte) (*ClientHandlerData, MessagePayload, error) {
	var messageType NetMessageType
	var rawPayload []byte

	switch c.payloadType {
	case PayloadTypeBinary:
		var msg BinaryNetPacket
		if err := msg.Unmarshal(buf); err != nil {
			return nil, nil, err
		}
		messageType = msg.Type
		rawPayload = msg.Payload
	case PayloadTypeText:
		var msg TextNetPacket
		if err := msg.Unmarshal(buf); err != nil {
			return nil, nil, err
		}
		messageType = msg.Type
		rawPayload = []byte(msg.Payload)
	default:
		return nil, nil, fmt.Errorf("unsupported payload type: %d", c.payloadType)
	}

	c.handlerMu.RLock()
	handler, ok := c.messageHandlers[messageType]
	c.handlerMu.RUnlock()
	if !ok {
		// 内置广播类型无 handler 时静默（服务端每次发布轨道都会广播，
		// 未订阅该通知属正常场景，刷 error 日志是噪音）
		if messageType == MsgTypeSignalTrackAvailable {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("message handler not found: %d", messageType)
	}

	if handler.Creator == nil {
		return handler, rawPayload, nil
	}

	payload := handler.Creator()
	if err := broker.Unmarshal(c.codec, rawPayload, &payload); err != nil {
		return nil, nil, err
	}
	return handler, payload, nil
}

func (c *Client) messageHandler(buf []byte) error {
	handler, payload, err := c.unmarshalMessage(buf)
	if err != nil {
		return err
	}
	if handler == nil || handler.Handler == nil {
		return nil
	}
	return handler.Handler(payload)
}

// AddLocalTrack 添加本地媒体轨道（音频/视频）
func (c *Client) AddLocalTrack(track *webrtc.TrackLocalStaticRTP) error {
	c.pcMu.RLock()
	pc := c.pc
	c.pcMu.RUnlock()

	if pc == nil {
		return errors.New("peer connection not established")
	}

	if _, err := pc.AddTrack(track); err != nil {
		return err
	}

	c.pcMu.Lock()
	c.localTracks[track.ID()] = track
	c.pcMu.Unlock()

	// 客户端是初始 offer 方：加轨后必须重新 offer，否则轨道不会真正协商发送
	if err := c.negotiate(); err != nil {
		return err
	}

	LogInfof("added local track: %s", track.ID())
	return nil
}

// negotiate 发起重协商：创建 Offer 发送给服务端，等待服务端经内置信令通道回填 Answer
func (c *Client) negotiate() error {
	c.pcMu.RLock()
	pc := c.pc
	c.pcMu.RUnlock()

	if pc == nil {
		return errors.New("peer connection not established")
	}

	offer, err := pc.CreateOffer(nil)
	if err != nil {
		return err
	}

	gatherDone := webrtc.GatheringCompletePromise(pc)
	if err = pc.SetLocalDescription(offer); err != nil {
		return err
	}
	select {
	case <-gatherDone:
	case <-time.After(c.signalTimeout):
		return errors.New("ice gathering timeout")
	}

	local := pc.LocalDescription()
	if local == nil {
		return errors.New("local description is nil")
	}

	return c.SendMessage(MsgTypeSignalRenegotiation, SignalRenegotiationMsg{
		Type:  "renegotiation",
		Offer: local,
	})
}

// handleSignalRenegotiation 处理服务端下发的重协商信令：
// - 携带 Offer：应用为远端描述并回 Answer（服务端新增下行轨道）
// - 携带 Answer：应用为远端描述（服务端确认客户端上行的 Offer）
func (c *Client) handleSignalRenegotiation(payload MessagePayload) error {
	msg, ok := payload.(*SignalRenegotiationMsg)
	if !ok || msg == nil {
		return errors.New("invalid renegotiation payload")
	}

	c.pcMu.RLock()
	pc := c.pc
	c.pcMu.RUnlock()

	if pc == nil {
		return errors.New("peer connection not established")
	}

	if msg.Answer != nil {
		if err := pc.SetRemoteDescription(*msg.Answer); err != nil {
			return err
		}
		gatherDone := webrtc.GatheringCompletePromise(pc)
		select {
		case <-gatherDone:
		case <-time.After(3 * time.Second):
		}
		return nil
	}

	if msg.Offer != nil {
		if err := pc.SetRemoteDescription(*msg.Offer); err != nil {
			return err
		}
		answer, err := pc.CreateAnswer(nil)
		if err != nil {
			return err
		}
		// 信令通道无 trickle，等 ICE 收敛把候选装进 SDP
		gatherDone := webrtc.GatheringCompletePromise(pc)
		if err = pc.SetLocalDescription(answer); err != nil {
			return err
		}
		select {
		case <-gatherDone:
		case <-time.After(c.signalTimeout):
			LogWarn("ice gathering timeout, sending current local description")
		}

		// gather 完成后取 pc.LocalDescription()（包含收集到的 ICE 候选）
		local := pc.LocalDescription()

		return c.SendMessage(MsgTypeSignalRenegotiation, SignalRenegotiationMsg{
			Type:   "renegotiation",
			Answer: local,
		})
	}

	return nil
}

// RemoveLocalTrack 移除本地媒体轨道
func (c *Client) RemoveLocalTrack(trackID string) {
	c.pcMu.Lock()
	defer c.pcMu.Unlock()

	if _, ok := c.localTracks[trackID]; ok {
		// TrackLocalStaticRTP 不需要手动关闭
		// 从 PeerConnection 中移除轨道（如果需要）
		delete(c.localTracks, trackID)
		LogInfof("removed local track: %s", trackID)
	}
}

// GetRemoteTracks 获取所有远程媒体轨道
func (c *Client) GetRemoteTracks() map[string]*webrtc.TrackRemote {
	c.pcMu.RLock()
	defer c.pcMu.RUnlock()

	result := make(map[string]*webrtc.TrackRemote)
	for k, v := range c.remoteTracks {
		result[k] = v
	}
	return result
}
