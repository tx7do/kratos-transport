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
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/tx7do/kratos-transport/broker"
)

type Binder func() Any

type ConnectHandler func(SessionID, bool)

type MessageHandler func(SessionID, MessagePayload) error

type RawMessageHandler func(SessionID, []byte) error

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
	return "tcp"
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

func (s *Server) DeregisterMessageHandler(messageType MessageType) {
	delete(s.messageHandlers, messageType)
}

// SendRawData send raw data to client
func (s *Server) SendRawData(sessionId SessionID, message []byte) error {
	session, ok := s.sessions[sessionId]
	if !ok {
		log.Error("[tcp] session not found:", sessionId)
		return errors.New(fmt.Sprintf("[tcp] session not found: %s", sessionId))
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
		log.Error("[tcp] marshal message exception:", err)
		return errors.New(fmt.Sprintf("[tcp] marshal message exception: %s", err.Error()))
	}

	return s.SendRawData(sessionId, buf)
}

func (s *Server) Broadcast(messageType MessageType, message MessagePayload) {
	buf, err := s.marshalMessage(messageType, message)
	if err != nil {
		log.Error(" [tcp] marshal message exception:", err)
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

	log.Infof("[tcp] server listening on: %s", s.lis.Addr().String())

	go s.run()

	go s.doAccept()

	return nil
}

func (s *Server) Stop(_ context.Context) error {
	log.Info("[tcp] server stopping")

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

func (s *Server) messageHandler(sessionId SessionID, buf []byte) error {
	if s.rawMessageHandler != nil {
		if err := s.rawMessageHandler(sessionId, buf); err != nil {
			log.Errorf("[tcp] raw data handler exception: %s", err)
			return err
		}
		return nil
	}

	var msg Message
	if err := msg.Unmarshal(buf); err != nil {
		log.Errorf("[tcp] decode message exception: %s", err)
		return err
	}

	handlerData, ok := s.messageHandlers[msg.Type]
	if !ok {
		log.Error("[tcp] message type not found:", msg.Type)
		return errors.New("message handler not found")
	}

	var payload MessagePayload

	if handlerData.Binder != nil {
		payload = handlerData.Binder()
	} else {
		payload = msg.Body
	}

	if err := broker.Unmarshal(s.codec, msg.Body, &payload); err != nil {
		log.Errorf("[tcp] unmarshal message exception: %s", err)
		return err
	}

	if err := handlerData.Handler(sessionId, payload); err != nil {
		log.Errorf("[tcp] message handler exception: %s", err)
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

func (s *Server) doAccept() {
	for {
		conn, err := s.lis.Accept()
		if err != nil {
			log.Error("[tcp] accept exception:", err)
			continue
		}

		session := NewSession(conn, s)
		session.server.register <- session

		session.Listen()
	}
}

func (s *Server) addSession(c *Session) {
	//log.Info("[tcp] add session: ", c.SessionID())
	s.sessions[c.SessionID()] = c

	if s.connectHandler != nil {
		s.connectHandler(c.SessionID(), true)
	}
}

func (s *Server) removeSession(c *Session) {
	for k, v := range s.sessions {
		if c == v {
			//log.Info("[tcp] remove session: ", c.SessionID())
			if s.connectHandler != nil {
				s.connectHandler(c.SessionID(), false)
			}
			delete(s.sessions, k)
			return
		}
	}
}
