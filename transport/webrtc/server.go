package webrtc

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
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

	var offer webrtc.SessionDescription
	if err := decodeSignalRequest(req, &offer); err != nil {
		res.WriteHeader(http.StatusBadRequest)
		_ = writeSignalError(res, err)
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

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		switch state {
		case webrtc.PeerConnectionStateClosed, webrtc.PeerConnectionStateFailed, webrtc.PeerConnectionStateDisconnected:
			session.Close()
		}
	})

	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		if !s.allowAnyDataLabel && s.dataChannelLabel != "" && dc.Label() != s.dataChannelLabel {
			_ = dc.Close()
			return
		}

		session.BindDataChannel(dc, func() {
			s.sessionManager.addSession(session)
			session.Listen()
		})
	})

	if err = pc.SetRemoteDescription(offer); err != nil {
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

func (s *Server) OnSessionAdded(session *Session) {
	if s.socketConnectHandler != nil && session != nil {
		s.socketConnectHandler(session.SessionID(), session.queries, true)
	}
}

func (s *Server) OnSessionRemoved(session *Session) {
	if s.socketConnectHandler != nil && session != nil {
		s.socketConnectHandler(session.SessionID(), session.queries, false)
	}
}
