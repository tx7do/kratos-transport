package websocket

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"sync"

	"github.com/go-kratos/kratos/v2/encoding"
	kratosTransport "github.com/go-kratos/kratos/v2/transport"

	ws "github.com/gorilla/websocket"

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
	upgrader *ws.Upgrader
	endpoint *url.URL

	network     string
	address     string
	path        string
	strictSlash bool
	injectToken bool
	tokenKey    string

	err   error
	codec encoding.Codec

	sessionManager *SessionManager

	register   chan *Session
	unregister chan *Session

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
		path:        "/",
		injectToken: true,
		tokenKey:    "token",

		messageHandlers: make(NetMessageHandlerMap),

		sessionManager: NewSessionManager(nil),

		upgrader: &ws.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},

		register:   make(chan *Session),
		unregister: make(chan *Session),

		payloadType: PayloadTypeBinary,
	}

	srv.sessionManager.RegisterObserver(srv)

	if err := srv.init(opts...); err != nil {
		LogError("websocket server init error:", err)
		return nil
	}

	return srv
}

func (s *Server) init(opts ...ServerOption) error {
	for _, o := range opts {
		o(s)
	}

	s.Server = &http.Server{
		TLSConfig: s.tlsConf,
	}

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

	http.HandleFunc(s.path, s.wsHandler)

	return s.err
}

func (s *Server) Name() string {
	return KindWebsocket
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
	var err error
	var buff []byte

	switch s.payloadType {
	case PayloadTypeBinary:
		var msg BinaryNetPacket
		msg.Type = messageType
		msg.Payload, err = broker.Marshal(s.codec, message)
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
		buf, err = broker.Marshal(s.codec, message)
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
	var err error
	var buf []byte

	switch s.payloadType {
	case PayloadTypeBinary:
		buf, err = s.marshalMessage(messageType, message)
		if err != nil {
			LogError("marshal binary message error:", err)
			return err
		}

		break

	case PayloadTypeText:
		buf, err = s.codec.Marshal(message)
		if err != nil {
			LogError("marshal text message error:", err)
			return err
		}

		break
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
	var err error
	var handler *MessageHandlerData
	var payload MessagePayload

	if handler, payload, err = s.unmarshalNetPacket(buf); err != nil {
		LogErrorf("unmarshal message failed: %s", err)
		return err
	}
	//LogDebug(payload)

	if err = handler.Handler(sessionId, payload); err != nil {
		LogErrorf("message handler failed: %s", err)
		return err
	}

	return nil
}

func (s *Server) wsHandler(res http.ResponseWriter, req *http.Request) {
	var token string
	if req.Header != nil {
		token = req.Header.Get("Sec-Websocket-Protocol")
	}

	if token != "" {
		s.upgrader.Subprotocols = []string{req.Header.Get("Sec-Websocket-Protocol")} //设置Sec-Websocket-Protocol
	}

	conn, err := s.upgrader.Upgrade(res, req, nil)
	if err != nil {
		LogError("upgrade exception:", err)
		return
	}

	vars := req.URL.Query()

	if token != "" {
		if s.injectToken {
			vars.Set(s.tokenKey, token)
		}
	}

	session := NewSession(s, conn, vars)
	session.Listen()
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

		s.endpoint = transport.NewRegistryEndpoint(KindWebsocket, addr)
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
	if s.running {
		return nil
	}

	if s.err = s.listenAndEndpoint(); s.err != nil {
		return s.err
	}

	if s.err != nil {
		return s.err
	}

	s.running = true

	s.BaseContext = func(net.Listener) context.Context {
		return ctx
	}
	LogInfof("server listening on: %s", s.lis.Addr().String())

	var err error
	if s.tlsConf != nil {
		err = s.ServeTLS(s.lis, "", "")
	} else {
		err = s.Serve(s.lis)
	}
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	if !s.running {
		return nil
	}

	LogInfo("server stopping...")

	err := s.Shutdown(ctx)
	s.err = nil

	s.running = false

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
