package tcp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/encoding"
	kratosTransport "github.com/go-kratos/kratos/v2/transport"
	"github.com/tx7do/kratos-transport/broker"
)

type Binder func() Any

type ConnectHandler func(SessionID, bool)

type MessageHandler func(SessionID, MessagePayload) error

type RawMessageHandler func(SessionID, []byte) (err error, msgType MessageType, msgBody []byte)

type HandlerData struct {
	Handler MessageHandler
	Binder  Binder
}
type MessageHandlerMap map[MessageType]HandlerData

var (
	_ kratosTransport.Server     = (*Server)(nil)
	_ kratosTransport.Endpointer = (*Server)(nil)
)

type Server struct {
	lis     net.Listener
	tlsConf *tls.Config

	network string
	address string

	timeout time.Duration

	err   error
	codec encoding.Codec

	messageHandlers   MessageHandlerMap
	rawMessageHandler RawMessageHandler
	connectHandler    ConnectHandler

	sessions   SessionMap
	register   chan *Session
	unregister chan *Session
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		network: "tcp",
		address: ":0",
		timeout: 1 * time.Second,

		messageHandlers: make(MessageHandlerMap),

		sessions: SessionMap{},

		register:   make(chan *Session),
		unregister: make(chan *Session),
	}

	srv.init(opts...)

	return srv
}

func (s *Server) Name() string {
	return string(KindTcp)
}

func (s *Server) Endpoint() (*url.URL, error) {
	addr := s.address

	prefix := "tcp://"
	if s.tlsConf == nil {
		if !strings.HasPrefix(addr, "tcp://") {
			prefix = "tcp://"
		}
	} else {
		if !strings.HasPrefix(addr, "tcp://") {
			prefix = "tcp://"
		}
	}
	addr = prefix + addr

	var endpoint *url.URL
	endpoint, s.err = url.Parse(addr)
	return endpoint, nil
}

func (s *Server) SessionCount() int {
	return len(s.sessions)
}

func (s *Server) RegisterMessageHandler(messageType MessageType, handler MessageHandler, binder Binder) {
	if _, ok := s.messageHandlers[messageType]; ok {
		return
	}

	s.messageHandlers[messageType] = HandlerData{
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

// SendRawData send raw data to client
func (s *Server) SendRawData(sessionId SessionID, message []byte) error {
	session, ok := s.sessions[sessionId]
	if !ok {
		LogError("session not found:", sessionId)
		return errors.New(fmt.Sprintf("session not found: %s", sessionId))
	}

	session.SendMessage(message)

	return nil
}

func (s *Server) BroadcastRawData(message []byte) {
	for _, c := range s.sessions {
		c.SendMessage(message)
	}
}

func (s *Server) SendMessage(sessionId SessionID, messageType MessageType, message MessagePayload) error {
	buf, err := s.marshalMessage(messageType, message)
	if err != nil {
		LogError("marshal message exception:", err)
		return errors.New(fmt.Sprintf("marshal message exception: %s", err.Error()))
	}

	return s.SendRawData(sessionId, buf)
}

func (s *Server) Broadcast(messageType MessageType, message MessagePayload) {
	buf, err := s.marshalMessage(messageType, message)
	if err != nil {
		LogError(" marshal message exception:", err)
		return
	}

	s.BroadcastRawData(buf)
}

func (s *Server) init(opts ...ServerOption) {
	for _, o := range opts {
		o(s)
	}
}

func (s *Server) Start(_ context.Context) error {
	if s.err = s.listen(); s.err != nil {
		return s.err
	}

	LogInfof("server listening on: %s", s.lis.Addr().String())

	go s.run()

	go s.doAccept()

	return nil
}

func (s *Server) Stop(_ context.Context) error {
	LogInfo("server stopping")

	if s.lis != nil {
		_ = s.lis.Close()
		s.lis = nil
	}

	return nil
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

// GetMessageHandler find message handler
func (s *Server) GetMessageHandler(msgType MessageType) (error, *HandlerData) {
	handlerData, ok := s.messageHandlers[msgType]
	if !ok {
		errMsg := fmt.Sprintf("[%d] message handler not found", msgType)
		LogError(errMsg)
		return errors.New(errMsg), nil
	}

	return nil, &handlerData
}

func (s *Server) HandleMessage(sessionId SessionID, msgType MessageType, msgBody []byte) error {
	var err error

	var handlerData *HandlerData
	if err, handlerData = s.GetMessageHandler(msgType); err != nil {
		return err
	}

	var payload MessagePayload
	if handlerData.Binder != nil {
		payload = handlerData.Binder()
	} else {
		payload = msgBody
	}

	if err = broker.Unmarshal(s.codec, msgBody, &payload); err != nil {
		LogErrorf("unmarshal message exception: %s", err)
		return err
	}

	if err = handlerData.Handler(sessionId, payload); err != nil {
		LogErrorf("message handler exception: %s", err)
		return err
	}

	return nil
}

// messageHandler socket data process
func (s *Server) messageHandler(sessionId SessionID, buf []byte) error {
	var err error

	if s.rawMessageHandler != nil {
		var msgType MessageType
		var msgBody []byte
		if err, msgType, msgBody = s.rawMessageHandler(sessionId, buf); err != nil {
			LogErrorf("raw data handler exception: %s", err)
			return err
		}

		if err = s.HandleMessage(sessionId, msgType, msgBody); err != nil {
			return err
		}

		return nil
	}

	var msg Message
	if err = msg.Unmarshal(buf); err != nil {
		LogErrorf("decode message exception: %s", err)
		return err
	}

	if err = s.HandleMessage(sessionId, msg.Type, msg.Body); err != nil {
		return err
	}

	return nil
}

func (s *Server) listen() error {
	if s.lis == nil {
		lis, err := net.Listen(s.network, s.address)
		if err != nil {
			return err
		}
		s.lis = lis
	}

	return nil
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

// doAccept accept connection handler
func (s *Server) doAccept() {
	for {
		if s.lis == nil {
			return
		}

		conn, err := s.lis.Accept()
		if err != nil {
			LogError("accept exception:", err)
			continue
		}

		session := NewSession(conn, s)
		session.server.register <- session

		session.Listen()
	}
}

func (s *Server) addSession(c *Session) {
	//LogInfo("add session: ", c.SessionID())
	s.sessions[c.SessionID()] = c

	if s.connectHandler != nil {
		s.connectHandler(c.SessionID(), true)
	}
}

func (s *Server) removeSession(c *Session) {
	for k, v := range s.sessions {
		if c == v {
			//LogInfo("remove session: ", c.SessionID())
			if s.connectHandler != nil {
				s.connectHandler(c.SessionID(), false)
			}
			delete(s.sessions, k)
			return
		}
	}
}
