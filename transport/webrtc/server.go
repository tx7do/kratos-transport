package webrtc

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/encoding"
	kratosTransport "github.com/go-kratos/kratos/v2/transport"
	"github.com/pion/webrtc/v4"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/transport"
)

var (
	_ kratosTransport.Server     = (*Server)(nil)
	_ kratosTransport.Endpointer = (*Server)(nil)
)

type Server struct {
	*http.Server

	lis      net.Listener
	tlsConf  *tls.Config
	endpoint *url.URL

	network     string
	address     string
	path        string
	strictSlash bool
	injectToken bool
	tokenKey    string
	checkOrigin func(*http.Request) bool
	enableCORS  bool

	corsAllowOrigin  string
	corsAllowMethods string
	corsAllowHeaders string

	err   error
	codec encoding.Codec

	webrtcAPI         *webrtc.API
	webrtcConfig      webrtc.Configuration
	dataChannelLabel  string
	allowAnyDataLabel bool

	sessionManager *SessionManager

	sfuRouter *SFURouter

	payloadType PayloadType

	messageHandlers NetMessageHandlerMap

	netPacketMarshaler   NetPacketMarshaler
	netPacketUnmarshaler NetPacketUnmarshaler

	socketConnectHandler SocketConnectHandler
	socketRawDataHandler SocketRawDataHandler

	running   bool
	stateMu   sync.RWMutex
	handlerMu sync.RWMutex
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		network:     "tcp",
		address:     ":0",
		strictSlash: true,
		path:        "/signal",
		injectToken: true,
		tokenKey:    "token",
		checkOrigin: func(_ *http.Request) bool { return true },
		enableCORS:  true,

		corsAllowOrigin:  "*",
		corsAllowMethods: "POST, OPTIONS",
		corsAllowHeaders: "Content-Type, Authorization",

		webrtcConfig: webrtc.Configuration{},

		dataChannelLabel:  "kratos",
		allowAnyDataLabel: true,

		messageHandlers: make(NetMessageHandlerMap),

		sessionManager: NewSessionManager(nil),

		sfuRouter: NewSFURouter(),

		payloadType: PayloadTypeBinary,
	}

	srv.sessionManager.RegisterObserver(srv)

	if err := srv.init(opts...); err != nil {
		LogError("webrtc server init error:", err)
		return nil
	}

	return srv
}

func (s *Server) init(opts ...ServerOption) error {

	// 内置信令处理器：处理客户端回传的重协商 Answer/Offer。
	// 在用户 opts 应用之后注册：RegisterMessageHandler 先注册者优先，
	// 因此用户无法覆盖内置信令处理（防误抢）
	s.RegisterMessageHandler(MsgTypeSignalRenegotiation,
		func(sessionId SessionID, payload MessagePayload) error {
			return s.handleSignalRenegotiation(sessionId, payload)
		},
		func() any {
			return &SignalRenegotiationMsg{}
		},
	)

	for _, o := range opts {
		o(s)
	}

	s.rebuildHTTPServer()

	if s.netPacketMarshaler == nil {
		s.netPacketMarshaler = s.defaultMarshalNetPacket
	}
	if s.netPacketUnmarshaler == nil {
		s.netPacketUnmarshaler = s.defaultUnmarshalNetPacket
	}

	if s.socketRawDataHandler == nil {
		s.socketRawDataHandler = s.defaultHandleSocketRawData
	}

	// rebuildHTTPServer 重建被 Shutdown 毒化的 http.Server（重启支持）

	if s.codec == nil {
		s.codec = encoding.GetCodec("json")
		if s.codec == nil {
			s.codec = encoding.GetCodec("bytes")
		}
	}

	return s.err
}

func (s *Server) Name() string {
	return KindWebRTC
}

func (s *Server) RegisterMessageHandler(messageType NetMessageType, handler NetMessageHandler, binder Creator) {
	s.handlerMu.Lock()
	defer s.handlerMu.Unlock()

	if _, ok := s.messageHandlers[messageType]; ok {
		return
	}

	s.messageHandlers[messageType] = &MessageHandlerData{
		handler, binder,
	}
}

func RegisterServerMessageHandler[T any](srv *Server, messageType NetMessageType, handler func(SessionID, *T) error) {
	srv.RegisterMessageHandler(messageType,
		func(sessionId SessionID, payload MessagePayload) error {
			switch t := payload.(type) {
			case *T:
				return handler(sessionId, t)
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

func (s *Server) DeregisterMessageHandler(messageType NetMessageType) {
	s.handlerMu.Lock()
	defer s.handlerMu.Unlock()

	delete(s.messageHandlers, messageType)
}

func (s *Server) GetMessageHandler(messageType NetMessageType) *MessageHandlerData {
	s.handlerMu.RLock()
	defer s.handlerMu.RUnlock()

	return s.messageHandlers[messageType]
}

func (s *Server) marshalMessage(messageType NetMessageType, message MessagePayload) ([]byte, error) {
	if s.netPacketMarshaler == nil {
		return s.defaultMarshalNetPacket(messageType, message)
	} else {
		return s.netPacketMarshaler(messageType, message)
	}
}

func (s *Server) defaultMarshalNetPacket(messageType NetMessageType, message MessagePayload) ([]byte, error) {
	switch s.payloadType {
	case PayloadTypeBinary:
		var msg BinaryNetPacket
		msg.Type = messageType
		payload, err := broker.Marshal(s.codec, message)
		if err != nil {
			return nil, err
		}
		msg.Payload = payload
		return msg.Marshal()

	case PayloadTypeText:
		var msg TextNetPacket
		msg.Type = messageType
		payload, err := broker.Marshal(s.codec, message)
		if err != nil {
			return nil, err
		}
		msg.Payload = string(payload)
		return msg.Marshal()
	}

	return nil, fmt.Errorf("unsupported payload type: %d", s.payloadType)
}

func (s *Server) SendRawMessage(sessionId SessionID, message []byte) error {
	session := s.sessionManager.getSession(sessionId)
	if session == nil {
		LogError("session not found:", sessionId)
		return errors.New("session not found")
	}

	session.SendMessage(message)

	return nil
}

func (s *Server) SendMessage(sessionId SessionID, messageType NetMessageType, message MessagePayload) error {
	buf, err := s.marshalMessage(messageType, message)
	if err != nil {
		LogError("marshal message error:", err)
		return err
	}

	return s.SendRawMessage(sessionId, buf)
}

func (s *Server) Broadcast(messageType NetMessageType, message MessagePayload) {
	buf, err := s.marshalMessage(messageType, message)
	if err != nil {
		LogError(" marshal message error:", err)
		return
	}

	s.sessionManager.rangeSessions(func(_ SessionID, session *Session) bool {
		session.SendMessage(buf)
		return true
	})
}

func (s *Server) unmarshalNetPacket(buf []byte) (*MessageHandlerData, MessagePayload, error) {
	if s.netPacketUnmarshaler != nil {
		return s.netPacketUnmarshaler(buf)
	} else {
		return s.defaultUnmarshalNetPacket(buf)
	}
}

func (s *Server) defaultUnmarshalNetPacket(buf []byte) (handler *MessageHandlerData, payload MessagePayload, err error) {
	var messageType NetMessageType
	var rawPayload []byte

	switch s.payloadType {
	case PayloadTypeBinary:
		var msg BinaryNetPacket
		if err = msg.Unmarshal(buf); err != nil {
			LogErrorf("decode message exception: %s", err)
			return nil, nil, err
		}
		messageType = msg.Type
		rawPayload = msg.Payload

	case PayloadTypeText:
		var msg TextNetPacket
		if err = msg.Unmarshal(buf); err != nil {
			LogErrorf("decode message exception: %s", err)
			return nil, nil, err
		}
		messageType = msg.Type
		rawPayload = []byte(msg.Payload)
	}

	if handler = s.GetMessageHandler(messageType); handler == nil {
		LogError("message handler not found:", messageType)
		return nil, nil, errors.New("message handler not found")
	}

	if payload = handler.Create(); payload == nil {
		payload = rawPayload
	} else {
		if err = broker.Unmarshal(s.codec, rawPayload, &payload); err != nil {
			LogErrorf("unmarshal message exception: %s", err)
			return nil, nil, err
		}
	}

	//LogDebug(string(rawPayload))

	return
}

// handleSocketRawData process raw data received from socket
func (s *Server) handleSocketRawData(sessionId SessionID, buf []byte) error {
	if s.socketRawDataHandler != nil {
		return s.socketRawDataHandler(sessionId, buf)
	} else {
		return s.defaultHandleSocketRawData(sessionId, buf)
	}
}

func (s *Server) defaultHandleSocketRawData(sessionId SessionID, buf []byte) error {
	handler, payload, err := s.unmarshalNetPacket(buf)
	if err != nil {
		LogErrorf("unmarshal message failed: %s", err)
		return err
	}

	if err = handler.Handler(sessionId, payload); err != nil {
		LogErrorf("message handler failed: %s", err)
		return err
	}

	return nil
}

func (s *Server) signalHandler(res http.ResponseWriter, req *http.Request) {
	if s.enableCORS {
		s.writeCORSHeaders(res, req)
	}

	if req.Method == http.MethodOptions {
		res.WriteHeader(http.StatusNoContent)
		return
	}

	if req.Method != http.MethodPost {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if s.checkOrigin != nil && !s.checkOrigin(req) {
		res.WriteHeader(http.StatusForbidden)
		return
	}

	// 解析信令请求（限制请求体大小，防止无限制读取）
	req.Body = http.MaxBytesReader(res, req.Body, 1<<20)
	body, err := io.ReadAll(req.Body)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		_ = writeSignalError(res, err)
		return
	}
	defer req.Body.Close()

	var signalReq signalRequest
	if err = json.Unmarshal(body, &signalReq); err != nil {
		res.WriteHeader(http.StatusBadRequest)
		_ = writeSignalError(res, err)
		return
	}

	// 处理 ICE Candidate
	if signalReq.Candidate != nil {
		s.handleICECandidate(signalReq.Candidate, res)
		return
	}

	// 处理 Offer
	if signalReq.Offer == nil || signalReq.Offer.Type == 0 || signalReq.Offer.SDP == "" {
		res.WriteHeader(http.StatusBadRequest)
		_ = writeSignalError(res, errors.New("invalid offer"))
		return
	}

	vars := req.URL.Query()
	token := req.Header.Get("Authorization")
	if token != "" && s.injectToken {
		vars.Set(s.tokenKey, token)
	}

	pc, err := s.newPeerConnection()
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		_ = writeSignalError(res, err)
		return
	}

	session := NewSession(s, pc, vars)

	// 设置 ICE Candidate 回调
	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		// 可以通过 WebSocket 或其他方式发送 candidate，这里暂不实现
		LogDebugf("ICE candidate generated for session %s", session.SessionID())
	})

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		LogInfof("session %s connection state changed: %s", session.SessionID(), state.String())
		switch state {
		case webrtc.PeerConnectionStateClosed, webrtc.PeerConnectionStateFailed, webrtc.PeerConnectionStateDisconnected:
			LogInfof("session %s closing due to state: %s", session.SessionID(), state.String())
			session.Close()
		}
	})

	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		LogInfof("session %s data channel opened: %s", session.SessionID(), dc.Label())
		if !s.allowAnyDataLabel && s.dataChannelLabel != "" && dc.Label() != s.dataChannelLabel {
			LogWarnf("session %s rejecting data channel with label: %s", session.SessionID(), dc.Label())
			_ = dc.Close()
			return
		}

		session.BindDataChannel(dc, func() {
			LogInfof("session %s added to session manager", session.SessionID())
			s.sessionManager.addSession(session)
			session.Listen()
		})
	})

	// 处理 incoming 媒体轨道
	pc.OnTrack(func(remote *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		LogInfof("session %s received track: kind=%s, codec=%s",
			session.SessionID(), remote.Kind(), remote.Codec().MimeType)

		// 添加到 SFU 路由器
		mediaTrack := s.sfuRouter.AddTrack(session.SessionID(), remote, receiver)

		// 通知其他客户端有新轨道可用（通过数据通道发送信令）
		s.broadcastTrackAvailable(session.SessionID(), mediaTrack)
	})

	if err = pc.SetRemoteDescription(*signalReq.Offer); err != nil {
		session.Close()
		res.WriteHeader(http.StatusBadRequest)
		_ = writeSignalError(res, err)
		return
	}

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		session.Close()
		res.WriteHeader(http.StatusInternalServerError)
		_ = writeSignalError(res, err)
		return
	}

	gatherDone := webrtc.GatheringCompletePromise(pc)
	if err = pc.SetLocalDescription(answer); err != nil {
		session.Close()
		res.WriteHeader(http.StatusInternalServerError)
		_ = writeSignalError(res, err)
		return
	}
	<-gatherDone

	local := pc.LocalDescription()
	if local == nil {
		session.Close()
		res.WriteHeader(http.StatusInternalServerError)
		_ = writeSignalError(res, errors.New("local description is nil"))
		return
	}

	res.Header().Set("Content-Type", "application/json")
	if err = encodeSignalResponse(res, &signalResponse{Answer: *local, SessionID: session.SessionID()}); err != nil {
		session.Close()
	}
}

func (s *Server) writeCORSHeaders(res http.ResponseWriter, _ *http.Request) {
	allowOrigin := s.corsAllowOrigin
	if allowOrigin == "" {
		allowOrigin = "*"
	}

	headers := res.Header()
	headers.Set("Access-Control-Allow-Origin", allowOrigin)
	headers.Set("Access-Control-Allow-Methods", s.corsAllowMethods)
	headers.Set("Access-Control-Allow-Headers", s.corsAllowHeaders)
	headers.Add("Vary", "Origin")
	headers.Add("Vary", "Access-Control-Request-Method")
	headers.Add("Vary", "Access-Control-Request-Headers")
}

func (s *Server) newPeerConnection() (*webrtc.PeerConnection, error) {
	if s.webrtcAPI != nil {
		return s.webrtcAPI.NewPeerConnection(s.webrtcConfig)
	}
	return webrtc.NewPeerConnection(s.webrtcConfig)
}

func (s *Server) listenAndEndpoint() error {
	if s.lis == nil {
		lis, err := net.Listen(s.network, s.address)
		if err != nil {
			return err
		}
		s.lis = lis
	}

	if s.endpoint == nil {
		// 如果传入的是完整的ip地址，则不需要调整。
		// 如果传入的只有端口号，则会调整为完整的地址，但，IP地址或许会不正确。
		addr, err := transport.AdjustAddress(s.address, s.lis)
		if err != nil {
			return err
		}

		s.endpoint = transport.NewRegistryEndpoint(KindWebRTC, addr)
		if s.endpoint != nil {
			s.endpoint.Path = s.path
		}
	}

	return nil
}

func (s *Server) Endpoint() (*url.URL, error) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	if err := s.listenAndEndpoint(); err != nil {
		return nil, err
	}
	return s.endpoint, nil
}

func (s *Server) Start(ctx context.Context) error {
	s.stateMu.Lock()
	if s.running {
		s.stateMu.Unlock()
		return nil
	}

	if s.err = s.listenAndEndpoint(); s.err != nil {
		s.stateMu.Unlock()
		return s.err
	}

	if s.err != nil {
		s.stateMu.Unlock()
		return s.err
	}

	lis := s.lis
	s.running = true
	s.stateMu.Unlock()

	s.BaseContext = func(net.Listener) context.Context {
		return ctx
	}
	LogInfof("server listening on: %s", lis.Addr().String())

	var err error
	if s.tlsConf != nil {
		err = s.ServeTLS(lis, "", "")
	} else {
		err = s.Serve(lis)
	}

	s.stateMu.Lock()
	s.running = false
	s.stateMu.Unlock()

	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.stateMu.Lock()
	if !s.running {
		s.stateMu.Unlock()
		return nil
	}
	s.stateMu.Unlock()

	LogInfo("server stopping...")

	err := s.Shutdown(ctx)
	s.sessionManager.closeAllAndWait()

	// 重建 http.Server 并释放 listener/endpoint，支持 Stop→Start 重启
	s.rebuildHTTPServer()
	if s.lis != nil {
		_ = s.lis.Close()
		s.lis = nil
	}
	s.endpoint = nil

	s.stateMu.Lock()
	s.err = nil
	s.running = false
	s.stateMu.Unlock()

	LogInfo("server stopped.")

	return err
}

// removeSession removes a session from the manager, used by SessionHooks.
func (s *Server) removeSession(session *Session) {
	if s.sessionManager == nil {
		return
	}
	s.sessionManager.removeSession(session)
}

func (s *Server) getPayloadType() PayloadType {
	return s.payloadType
}

// handleICECandidate 处理 ICE Candidate
func (s *Server) handleICECandidate(candidate *webrtc.ICECandidateInit, res http.ResponseWriter) {
	// TODO: 需要 session ID 来找到对应的 PeerConnection
	// 这里简化处理，实际需要通过 WebSocket 或其他方式维护 session 映射
	res.WriteHeader(http.StatusOK)
	res.Write([]byte(`{"status":"ok"}`))
	LogDebug("ICE candidate received")
}

func (s *Server) OnSessionAdded(session *Session) {
	if s.socketConnectHandler != nil && session != nil {
		s.socketConnectHandler(session.SessionID(), session.queries, true)
	}
}

func (s *Server) OnSessionRemoved(session *Session) {
	if s.socketConnectHandler != nil && session != nil {
		s.socketConnectHandler(session.SessionID(), session.queries, false)
	}

	// 清理 SFU 轨道
	if s.sfuRouter != nil {
		s.sfuRouter.RemoveSessionTracks(session.SessionID())
	}

	LogInfof("✓ session %s removed from session manager. Active sessions: %d",
		session.SessionID(), s.sessionManager.count())
}

// SubscribeToPublisher 订阅指定发布者的媒体流
func (s *Server) SubscribeToPublisher(subscriberID SessionID, publisherID SessionID) error {
	session := s.sessionManager.getSession(subscriberID)
	if session == nil {
		return errors.New("subscriber session not found")
	}

	pc := session.PeerConnection()
	if pc == nil {
		return errors.New("peer connection not found")
	}

	// 记录订阅关系（否则 UnsubscribeFromPublisher 查表为空、退订是空操作），
	// 并获取发布者的所有轨道
	tracks := s.sfuRouter.Subscribe(subscriberID, publisherID)
	if len(tracks) == 0 {
		LogWarnf("no tracks available from publisher %s", publisherID)
		return nil
	}

	// 为每个轨道创建本地轨道并添加到 PeerConnection
	added := 0
	for _, mediaTrack := range tracks {
		localTrack, err := s.sfuRouter.CreateLocalTrackForSubscriber(subscriberID, mediaTrack, pc)
		if err != nil {
			LogErrorf("create local track error: %s", err)
			continue
		}

		_ = localTrack // 保持引用
		added++
	}

	// 全部轨道加完后再统一发一次重协商 Offer（服务端作为 offer 方，
	// stable 状态下 CreateAnswer 是非法的，这正是旧实现失败的原因）
	if added > 0 {
		if err := s.sendRenegotiation(session, pc); err != nil {
			LogErrorf("send renegotiation error: %s", err)
		}
	}

	LogInfof("session subscribed to publisher %s (%d tracks)", subscriberID, len(tracks))
	return nil
}

// UnsubscribeFromPublisher 取消订阅
func (s *Server) UnsubscribeFromPublisher(subscriberID SessionID, publisherID SessionID) {
	s.sfuRouter.Unsubscribe(subscriberID, publisherID)
	LogInfof("session %s unsubscribed from publisher %s", subscriberID, publisherID)
}

// broadcastTrackAvailable 广播轨道可用通知
// 信令经 BinaryNetPacket（MsgTypeSignalTrackAvailable）封包，
// 客户端按标准消息分发即可接收（此前裸 JSON 无法通过客户端解析）
func (s *Server) broadcastTrackAvailable(publisherID SessionID, track *MediaTrack) {
	msg := SignalTrackAvailableMsg{
		Type:        "track_available",
		PublisherID: string(publisherID),
		TrackID:     track.ID(),
		Kind:        track.Kind().String(),
		Codec:       track.Codec().MimeType,
	}

	payload, err := broker.Marshal(s.codec, msg)
	if err != nil {
		LogErrorf("marshal track available message error: %s", err)
		return
	}

	pkt, err := (&BinaryNetPacket{Type: MsgTypeSignalTrackAvailable, Payload: payload}).Marshal()
	if err != nil {
		LogErrorf("marshal track available packet error: %s", err)
		return
	}

	// 发送给所有其他客户端
	s.sessionManager.rangeSessions(func(sessionID SessionID, session *Session) bool {
		if sessionID != publisherID {
			// 通过数据通道发送信令
			session.SendMessage(pkt)
		}
		return true
	})
}

// sendRenegotiation 发送重新协商信令（服务端作为 offer 方）：
// 旧实现在 stable 状态调 CreateAnswer（pion 语义非法，必返回 ErrInvalidState），
// 且裸 JSON 不经过客户端的 BinaryNetPacket 分发，订阅链路端到端不通
func (s *Server) sendRenegotiation(session *Session, pc *webrtc.PeerConnection) error {
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		return err
	}

	gatherDone := webrtc.GatheringCompletePromise(pc)
	if err = pc.SetLocalDescription(offer); err != nil {
		return err
	}
	// PC 被关闭时 gathering promise 可能永不完成，加超时防 SubscribeToPublisher 无限阻塞
	select {
	case <-gatherDone:
	case <-time.After(5 * time.Second):
		LogWarn("ice gathering timeout, sending current local description")
	}

	local := pc.LocalDescription()
	if local == nil {
		return errors.New("local description is nil")
	}

	msg := SignalRenegotiationMsg{
		Type:      "renegotiation",
		SessionID: session.SessionID(),
		Offer:     local,
	}

	payload, err := broker.Marshal(s.codec, msg)
	if err != nil {
		return err
	}

	pkt, err := (&BinaryNetPacket{Type: MsgTypeSignalRenegotiation, Payload: payload}).Marshal()
	if err != nil {
		return err
	}

	session.SendMessage(pkt)
	return nil
}

// handleSignalRenegotiation 处理客户端回传的重协商信令（内置，不可被用户覆盖）：
// - 携带 Answer：应用为远端描述（客户端确认服务端下行的 Offer）
// - 携带 Offer：客户端要新增上行轨道，服务端应答
func (s *Server) handleSignalRenegotiation(sessionId SessionID, payload MessagePayload) error {
	msg, ok := payload.(*SignalRenegotiationMsg)
	if !ok || msg == nil {
		return errors.New("invalid renegotiation payload")
	}

	session := s.sessionManager.getSession(sessionId)
	if session == nil {
		return errors.New("session not found")
	}
	pc := session.PeerConnection()
	if pc == nil {
		return errors.New("peer connection not found")
	}

	if msg.Answer != nil {
		// Answer 方向无需 gather：候选由对端 Offer 侧收集并携带
		return pc.SetRemoteDescription(*msg.Answer)
	}

	if msg.Offer != nil {
		if err := pc.SetRemoteDescription(*msg.Offer); err != nil {
			return err
		}
		answer, err := pc.CreateAnswer(nil)
		if err != nil {
			return err
		}
		gatherDone := webrtc.GatheringCompletePromise(pc)
		if err = pc.SetLocalDescription(answer); err != nil {
			return err
		}
		// 信令通道无 trickle，必须等 ICE 收敛把候选装进 SDP
		select {
		case <-gatherDone:
		case <-time.After(5 * time.Second):
			LogWarn("ice gathering timeout, sending current local description")
		}

		reply := SignalRenegotiationMsg{
			Type:      "renegotiation",
			SessionID: sessionId,
			Answer:    &answer,
		}
		raw, err := broker.Marshal(s.codec, reply)
		if err != nil {
			return err
		}
		pkt, err := (&BinaryNetPacket{Type: MsgTypeSignalRenegotiation, Payload: raw}).Marshal()
		if err != nil {
			return err
		}
		session.SendMessage(pkt)
	}

	return nil
}

// rebuildHTTPServer 重建内嵌的 http.Server（Shutdown 后不可复用，重启时需要新实例）
func (s *Server) rebuildHTTPServer() {
	mux := http.NewServeMux()
	mux.HandleFunc(s.path, s.signalHandler)

	s.Server = &http.Server{TLSConfig: s.tlsConf, Handler: mux}
}
