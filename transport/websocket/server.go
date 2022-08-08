package websocket

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
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

type Any interface{}
type MessageType uint32
type MessagePayload Any

type Message struct {
	Type MessageType
	Body []byte
}

func (m *Message) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint32(m.Type)); err != nil {
		return nil, err
	}
	buf.Write(m.Body)
	return buf.Bytes(), nil
}

func (m *Message) Unmarshal(buf []byte) error {
	network := new(bytes.Buffer)
	network.Write(buf)

	if err := binary.Read(network, binary.LittleEndian, &m.Type); err != nil {
		return err
	}

	m.Body = network.Bytes()

	return nil
}

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
	endpoint *url.URL
	upgrader *ws.Upgrader

	strictSlash bool
	network     string
	address     string
	timeout     time.Duration
	path        string

	err   error
	log   *log.Helper
	Codec encoding.Codec

	messageHandlers MessageHandlerMap
	connectHandler  ConnectHandler

	sessions   SessionMap
	register   chan *Session
	unregister chan *Session
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		network:     "tcp",
		address:     ":0",
		timeout:     1 * time.Second,
		strictSlash: true,
		log:         log.NewHelper(log.GetLogger(), log.WithMessageKey("[websocket]")),

		messageHandlers: make(MessageHandlerMap),

		sessions: SessionMap{},
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
	return "websocket"
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
	return len(s.sessions)
}

func (s *Server) RegisterMessageHandler(messageType MessageType, handler MessageHandler, binder Binder) {
	s.messageHandlers[messageType] = HandlerData{handler, binder}
}

func (s *Server) DeregisterMessageHandler(messageType MessageType) {
	delete(s.messageHandlers, messageType)
}

func (s *Server) marshalMessage(messageType MessageType, message MessagePayload) ([]byte, error) {
	var err error
	var msg Message
	msg.Type = messageType
	msg.Body, err = broker.Marshal(s.Codec, message)
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
	c, ok := s.sessions[sessionId]
	if !ok {
		s.log.Error("session not found:", sessionId)
		return
	}

	buf, err := s.marshalMessage(messageType, message)
	if err != nil {
		s.log.Error("marshal message exception:", err)
		return
	}

	c.SendMessage(buf)
}

func (s *Server) Broadcast(messageType MessageType, message MessagePayload) {
	buf, err := s.marshalMessage(messageType, message)
	if err != nil {
		s.log.Error("marshal message exception:", err)
		return
	}

	for _, c := range s.sessions {
		c.SendMessage(buf)
	}
}

func (s *Server) messageHandler(sessionId SessionID, buf []byte) error {
	var msg Message
	if err := msg.Unmarshal(buf); err != nil {
		s.log.Errorf("decode message exception: %s", err)
		return err
	}

	handlerData, ok := s.messageHandlers[msg.Type]
	if !ok {
		s.log.Error("message type not found:", msg.Type)
		return errors.New("message handler not found")
	}

	var payload MessagePayload

	if handlerData.Binder != nil {
		payload = handlerData.Binder()
	}

	if err := broker.Unmarshal(s.Codec, msg.Body, payload); err != nil {
		s.log.Errorf("unmarshal message exception: %s", err)
		return err
	}

	if err := handlerData.Handler(sessionId, payload); err != nil {
		s.log.Errorf("message handler exception: %s", err)
		return err
	}

	return nil
}

func (s *Server) wsHandler(res http.ResponseWriter, req *http.Request) {
	conn, err := s.upgrader.Upgrade(res, req, nil)
	if err != nil {
		s.log.Error("upgrade exception:", err)
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
			return err
		}
		s.lis = lis
	}

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

	s.endpoint, s.err = url.Parse(addr)

	return nil
}

func (s *Server) Endpoint() (*url.URL, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.endpoint, nil
}

func (s *Server) run() {
	for {
		select {
		case client := <-s.register:
			s.addSession(client)
		case client := <-s.unregister:
			s.removeSession(client)
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
	s.log.Infof("server listening on: %s", s.lis.Addr().String())

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
	s.log.Info("server stopping")
	return s.Shutdown(ctx)
}

func (s *Server) addSession(c *Session) {
	//s.log.Info("add session: ", c.SessionID())
	s.sessions[c.SessionID()] = c

	if s.connectHandler != nil {
		s.connectHandler(c.SessionID(), true)
	}
}

func (s *Server) removeSession(c *Session) {
	for k, v := range s.sessions {
		if c == v {
			//s.log.Info("remove session: ", c.SessionID())
			if s.connectHandler != nil {
				s.connectHandler(c.SessionID(), false)
			}
			delete(s.sessions, k)
			return
		}
	}
}
