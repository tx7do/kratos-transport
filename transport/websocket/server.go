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
	"time"

	"github.com/go-kratos/kratos/v2/encoding"
	kratosTransport "github.com/go-kratos/kratos/v2/transport"

	ws "github.com/gorilla/websocket"

	"github.com/tx7do/kratos-transport/broker"
)

type Binder func() Any

type ConnectHandler func(sessionId SessionID, queries url.Values, connect bool)

type MessageHandler func(SessionID, MessagePayload) error

type HandlerData struct {
	Handler MessageHandler
	Binder  Binder
}
type MessageHandlerMap map[MessageType]*HandlerData

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

	timeout time.Duration

	err   error
	codec encoding.Codec

	messageHandlers MessageHandlerMap

	sessionMgr *SessionManager

	register   chan *Session
	unregister chan *Session

	payloadType PayloadType
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		network:     "tcp",
		address:     ":0",
		timeout:     1 * time.Second,
		strictSlash: true,
		path:        "/",
		injectToken: true,
		tokenKey:    "token",

		messageHandlers: make(MessageHandlerMap),

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
		panic("init websocket server failed")
	}

	return srv
}

func (s *Server) Name() string {
	return string(KindWebsocket)
}

func (s *Server) init(opts ...ServerOption) error {
	for _, o := range opts {
		o(s)
	}

	s.Server = &http.Server{
		TLSConfig: s.tlsConf,
	}

	http.HandleFunc(s.path, s.wsHandler)

	s.err = s.listen()

	return s.err
}

func (s *Server) SessionCount() int {
	return s.sessionMgr.Count()
}

func (s *Server) RegisterMessageHandler(messageType MessageType, handler MessageHandler, binder Binder) {
	if _, ok := s.messageHandlers[messageType]; ok {
		return
	}

	s.messageHandlers[messageType] = &HandlerData{
		handler, binder,
	}
}

func RegisterServerMessageHandler[T any](srv *Server, messageType MessageType, handler func(SessionID, *T) error) {
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
		func() Any {
			var t T
			return &t
		},
	)
}

func (s *Server) DeregisterMessageHandler(messageType MessageType) {
	delete(s.messageHandlers, messageType)
}

func (s *Server) marshalMessage(messageType MessageType, message MessagePayload) ([]byte, error) {
	var err error
	var buff []byte

	switch s.payloadType {
	case PayloadTypeBinary:
		var msg BinaryMessage
		msg.Type = messageType
		msg.Body, err = broker.Marshal(s.codec, message)
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
		var msg TextMessage
		msg.Type = messageType
		buf, err = broker.Marshal(s.codec, message)
		msg.Body = string(buf)
		if err != nil {
			return nil, err
		}
		buff, err = json.Marshal(msg)
		if err != nil {
			return nil, err
		}
		break
	}

	//LogInfo("marshalMessage:", string(buff))

	return buff, nil
}

func (s *Server) SendMessage(sessionId SessionID, messageType MessageType, message MessagePayload) {
	c, ok := s.sessionMgr.Get(sessionId)
	if !ok {
		LogError("session not found:", sessionId)
		return
	}

	switch s.payloadType {
	case PayloadTypeBinary:
		buf, err := s.marshalMessage(messageType, message)
		if err != nil {
			LogError("marshal message exception:", err)
			return
		}

		c.SendMessage(buf)
		break

	case PayloadTypeText:
		buf, err := s.codec.Marshal(message)
		if err != nil {
			LogError("marshal message exception:", err)
			return
		}

		c.SendMessage(buf)
		break
	}

}

func (s *Server) Broadcast(messageType MessageType, message MessagePayload) {
	buf, err := s.marshalMessage(messageType, message)
	if err != nil {
		LogError(" marshal message exception:", err)
		return
	}

	s.sessionMgr.Range(func(session *Session) {
		session.SendMessage(buf)
	})
}

func (s *Server) unmarshalMessage(buf []byte) (*HandlerData, MessagePayload, error) {
	var handler *HandlerData
	var payload MessagePayload

	switch s.payloadType {
	case PayloadTypeBinary:
		var msg BinaryMessage
		if err := msg.Unmarshal(buf); err != nil {
			LogErrorf("decode message exception: %s", err)
			return nil, nil, err
		}

		var ok bool
		handler, ok = s.messageHandlers[msg.Type]
		if !ok {
			LogError("message handler not found:", msg.Type)
			return nil, nil, errors.New("message handler not found")
		}

		if handler.Binder != nil {
			payload = handler.Binder()
		} else {
			payload = msg.Body
		}

		if err := broker.Unmarshal(s.codec, msg.Body, &payload); err != nil {
			LogErrorf("unmarshal message exception: %s", err)
			return nil, nil, err
		}
		//LogDebug(string(msg.Body))

	case PayloadTypeText:
		var msg TextMessage
		if err := msg.Unmarshal(buf); err != nil {
			LogErrorf("decode message exception: %s", err)
			return nil, nil, err
		}

		var ok bool
		handler, ok = s.messageHandlers[msg.Type]
		if !ok {
			LogError("message handler not found:", msg.Type)
			return nil, nil, errors.New("message handler not found")
		}

		if handler.Binder != nil {
			payload = handler.Binder()
		} else {
			payload = msg.Body
		}

		if err := broker.Unmarshal(s.codec, []byte(msg.Body), &payload); err != nil {
			LogErrorf("unmarshal message exception: %s", err)
			return nil, nil, err
		}
		//LogDebug(string(msg.Body))
	}

	return handler, payload, nil
}

func (s *Server) messageHandler(sessionId SessionID, buf []byte) error {
	var err error
	var handler *HandlerData
	var payload MessagePayload

	if handler, payload, err = s.unmarshalMessage(buf); err != nil {
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
