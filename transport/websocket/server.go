package websocket

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-kratos/kratos/v2/encoding"
	kratosTransport "github.com/go-kratos/kratos/v2/transport"

	ws "github.com/gorilla/websocket"

	"github.com/tx7do/kratos-transport/broker"
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

	network     string
	address     string
	path        string
	strictSlash bool
	injectToken bool
	tokenKey    string

	err   error
	codec encoding.Codec

	sessionMgr *SessionManager

	register   chan *Session
	unregister chan *Session

	payloadType PayloadType

	messageHandlers NetMessageHandlerMap

	netPacketMarshaler   NetPacketMarshaler
	netPacketUnmarshaler NetPacketUnmarshaler

	socketRawDataHandler SocketRawDataHandler
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

		sessionMgr: NewSessionManager(),
		upgrader: &ws.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},

		register:   make(chan *Session),
		unregister: make(chan *Session),

		payloadType: PayloadTypeBinary,
	}

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

	s.err = s.listen()

	return s.err
}

func (s *Server) Name() string {
	return string(KindWebsocket)
}

func (s *Server) SessionCount() int {
	return s.sessionMgr.Count()
}

func (s *Server) RegisterMessageHandler(messageType NetMessageType, handler NetMessageHandler, binder Creator) {
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
	delete(s.messageHandlers, messageType)
}

func (s *Server) GetMessageHandler(messageType NetMessageType) *MessageHandlerData {
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
	c, ok := s.sessionMgr.Get(sessionId)
	if !ok {
		LogError("session not found:", sessionId)
		return errors.New("session not found")
	}

	c.SendMessage(message)

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

	s.sessionMgr.Range(func(session *Session) {
		session.SendMessage(buf)
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
	}

	if err = broker.Unmarshal(s.codec, rawPayload, &payload); err != nil {
		LogErrorf("unmarshal message exception: %s", err)
		return nil, nil, err
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
	session.server.register <- session

	session.Listen()
}

func (s *Server) listen() error {
	if s.lis == nil {
		lis, err := net.Listen(s.network, s.address)
		if err != nil {
			s.err = err
			return err
		}
		s.lis = lis
	}

	return nil
}

func (s *Server) Endpoint() (*url.URL, error) {
	addr := s.address

	prefix := "ws://"
	if s.tlsConf == nil {
		if !strings.HasPrefix(addr, "ws://") {
			prefix = "ws://"
		}
	} else {
		if !strings.HasPrefix(addr, "wss://") {
			prefix = "wss://"
		}
	}
	addr = prefix + addr

	var endpoint *url.URL
	endpoint, s.err = url.Parse(addr)
	return endpoint, nil
}

func (s *Server) run() {
	for {
		select {
		case client := <-s.register:
			s.sessionMgr.Add(client)
		case client := <-s.unregister:
			s.sessionMgr.Remove(client)
		}
	}
}

func (s *Server) Start(ctx context.Context) error {
	if s.err != nil {
		return s.err
	}
	s.BaseContext = func(net.Listener) context.Context {
		return ctx
	}
	LogInfof("server listening on: %s", s.lis.Addr().String())

	go s.run()

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
	LogInfo("server stopping")
	return s.Shutdown(ctx)
}
