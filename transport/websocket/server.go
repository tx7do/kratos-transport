package websocket

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
	ws "github.com/gorilla/websocket"
	"github.com/tx7do/kratos-transport/broker"
)

type Binder func() Any

type ConnectHandler func(SessionID, bool)

type MessageHandler func(SessionID, MessagePayload) error

type HandlerData struct {
	Handler MessageHandler
	Binder  Binder
}
type MessageHandlerMap map[MessageType]HandlerData

var (
	_ transport.Server     = (*Server)(nil)
	_ transport.Endpointer = (*Server)(nil)
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

	timeout time.Duration

	err   error
	codec encoding.Codec

	messageHandlers MessageHandlerMap

	sessionMgr *SessionManager

	register   chan *Session
	unregister chan *Session
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		network:     "tcp",
		address:     ":0",
		timeout:     1 * time.Second,
		strictSlash: true,
		path:        "/",

		messageHandlers: make(MessageHandlerMap),

		sessionMgr: NewSessionManager(),
		upgrader: &ws.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},

		register:   make(chan *Session),
		unregister: make(chan *Session),
	}

	srv.init(opts...)

	srv.err = srv.listen()

	return srv
}

func (s *Server) Name() string {
	return string(KindWebsocket)
}

func (s *Server) init(opts ...ServerOption) {
	for _, o := range opts {
		o(s)
	}

	s.Server = &http.Server{
		TLSConfig: s.tlsConf,
	}

	http.HandleFunc(s.path, s.wsHandler)
}

func (s *Server) SessionCount() int {
	return s.sessionMgr.Count()
}

func (s *Server) RegisterMessageHandler(messageType MessageType, handler MessageHandler, binder Binder) {
	if _, ok := s.messageHandlers[messageType]; ok {
		return
	}

	s.messageHandlers[messageType] = HandlerData{
		handler, binder,
	}
}

func (s *Server) DeregisterMessageHandler(messageType MessageType) {
	delete(s.messageHandlers, messageType)
}

func (s *Server) marshalMessage(messageType MessageType, message MessagePayload) ([]byte, error) {
	var err error
	var msg Message
	msg.Type = messageType
	msg.Body, err = broker.Marshal(s.codec, message)
	if err != nil {
		return nil, err
	}

	buff, err := msg.Marshal()
	if err != nil {
		return nil, err
	}

	return buff, nil
}

func (s *Server) SendMessage(sessionId SessionID, messageType MessageType, message MessagePayload) {
	c, ok := s.sessionMgr.Get(sessionId)
	if !ok {
		log.Error("[websocket] session not found:", sessionId)
		return
	}

	switch messageType {
	case ws.BinaryMessage:
		buf, err := s.marshalMessage(messageType, message)
		if err != nil {
			log.Error("[websocket] marshal message exception:", err)
			return
		}

		c.SendMessage(buf)
		break
	case ws.TextMessage:
		buf, err := s.codec.Marshal(message)
		if err != nil {
			log.Error("[websocket] marshal message exception:", err)
			return
		}

		c.sendTextMessage(string(buf))
		break
	}

}

func (s *Server) Broadcast(messageType MessageType, message MessagePayload) {
	switch messageType {
	case ws.BinaryMessage:
		buf, err := s.marshalMessage(messageType, message)
		if err != nil {
			log.Error(" [websocket] marshal message exception:", err)
			return
		}

		s.sessionMgr.Range(func(session *Session) {
			session.SendMessage(buf)
		})
		break
	case ws.TextMessage:
		buf, err := s.codec.Marshal(message)
		if err != nil {
			log.Error("[websocket] marshal message exception:", err)
			return
		}

		s.sessionMgr.Range(func(session *Session) {
			session.sendTextMessage(string(buf))
		})
		break
	}

}

func (s *Server) messageHandler(sessionId SessionID, messageType MessageType, buf []byte) error {
	var msg Message
	if err := msg.Unmarshal(buf); err != nil {
		log.Errorf("[websocket] decode message exception: %s", err)
		return err
	}
	msg.Type = messageType
	handlerData, ok := s.messageHandlers[msg.Type]
	if !ok {
		log.Error("[websocket] message type not found:", msg.Type)
		return errors.New("message handler not found")
	}

	var payload MessagePayload

	switch messageType {
	case ws.TextMessage:
		msg.Body = buf
		break
	}
	if handlerData.Binder != nil {
		payload = handlerData.Binder()
	} else {
		payload = msg.Body
	}

	if err := broker.Unmarshal(s.codec, msg.Body, &payload); err != nil {
		log.Errorf("[websocket] unmarshal message exception: %s", err)
		return err
	}
	// log.Debug(msg.Body)
	if err := handlerData.Handler(sessionId, msg.Body); err != nil {
		log.Errorf("[websocket] message handler exception: %s", err)
		return err
	}
	return nil
}

func (s *Server) wsHandler(res http.ResponseWriter, req *http.Request) {
	conn, err := s.upgrader.Upgrade(res, req, nil)
	if err != nil {
		log.Error("[websocket] upgrade exception:", err)
		return
	}

	session := NewSession(conn, s)
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
	log.Infof("[websocket] server listening on: %s", s.lis.Addr().String())

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
	log.Info("[websocket] server stopping")
	return s.Shutdown(ctx)
}
