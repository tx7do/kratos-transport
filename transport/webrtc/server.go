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
	for _, o := range opts {
		o(s)
	}

	mux := http.NewServeMux()
	mux.HandleFunc(s.path, s.signalHandler)

	s.Server = &http.Server{TLSConfig: s.tlsConf, Handler: mux}

	if s.netPacketMarshaler == nil {
		s.netPacketMarshaler = s.defaultMarshalNetPacket
	}
	if s.netPacketUnmarshaler == nil {
		s.netPacketUnmarshaler = s.defaultUnmarshalNetPacket
	}

	if s.socketRawDataHandler == nil {
		s.socketRawDataHandler = s.defaultHandleSocketRawData
	}

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

	// 解析信令请求
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

	// 获取发布者的所有轨道
	tracks := s.sfuRouter.GetPublisherTracks(publisherID)
	if len(tracks) == 0 {
		LogWarnf("no tracks available from publisher %s", publisherID)
		return nil
	}

	// 为每个轨道创建本地轨道并添加到 PeerConnection
	for _, mediaTrack := range tracks {
		localTrack, err := s.sfuRouter.CreateLocalTrackForSubscriber(subscriberID, mediaTrack, pc)
		if err != nil {
			LogErrorf("create local track error: %s", err)
			continue
		}

		// 发送 renegotation 信令
		if err := s.sendRenegotiation(session, pc); err != nil {
			LogErrorf("send renegotiation error: %s", err)
		}

		_ = localTrack // 保持引用
	}

	LogInfof("session %s subscribed to publisher %s (%d tracks)", subscriberID, publisherID, len(tracks))
	return nil
}

// UnsubscribeFromPublisher 取消订阅
func (s *Server) UnsubscribeFromPublisher(subscriberID SessionID, publisherID SessionID) {
	s.sfuRouter.Unsubscribe(subscriberID, publisherID)
	LogInfof("session %s unsubscribed from publisher %s", subscriberID, publisherID)
}

// broadcastTrackAvailable 广播轨道可用通知
func (s *Server) broadcastTrackAvailable(publisherID SessionID, track *MediaTrack) {
	// 构造信令消息
	type TrackAvailableMsg struct {
		Type        string `json:"type"`
		PublisherID string `json:"publisher_id"`
		TrackID     string `json:"track_id"`
		Kind        string `json:"kind"`
		Codec       string `json:"codec"`
	}

	msg := TrackAvailableMsg{
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

	// 发送给所有其他客户端
	s.sessionManager.rangeSessions(func(sessionID SessionID, session *Session) bool {
		if sessionID != publisherID {
			// 通过数据通道发送信令
			session.SendMessage(payload)
		}
		return true
	})
}

// sendRenegotiation 发送重新协商信令
func (s *Server) sendRenegotiation(session *Session, pc *webrtc.PeerConnection) error {
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		return err
	}

	gatherDone := webrtc.GatheringCompletePromise(pc)
	if err = pc.SetLocalDescription(answer); err != nil {
		return err
	}
	<-gatherDone

	local := pc.LocalDescription()
	if local == nil {
		return errors.New("local description is nil")
	}

	// 构造 renegotation 信令
	type RenegotiationMsg struct {
		Type      string                    `json:"type"`
		Answer    webrtc.SessionDescription `json:"answer"`
		SessionID SessionID                 `json:"session_id"`
	}

	msg := RenegotiationMsg{
		Type:      "renegotiation",
		Answer:    *local,
		SessionID: session.SessionID(),
	}

	payload, err := broker.Marshal(s.codec, msg)
	if err != nil {
		return err
	}

	session.SendMessage(payload)
	return nil
}
