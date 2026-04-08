package tcp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/encoding"
	kratosTransport "github.com/go-kratos/kratos/v2/transport"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/transport"
)

var (
	_ kratosTransport.Server     = (*Server)(nil)
	_ kratosTransport.Endpointer = (*Server)(nil)
)

type Server struct {
	lis      net.Listener
	tlsConf  *tls.Config
	endpoint *url.URL

	network string
	address string

	timeout time.Duration

	err   error
	codec encoding.Codec

	messageHandlers NetMessageHandlerMap

	socketConnectHandler SocketConnectHandler
	socketRawDataHandler SocketRawDataHandler

	netPacketMarshaler   NetPacketMarshaler
	netPacketUnmarshaler NetPacketUnmarshaler

	sessionManager *SessionManager

	running   bool
	stateMu   sync.RWMutex
	handlerMu sync.RWMutex
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		network: "tcp",
		address: ":0",
		timeout: 1 * time.Second,

		messageHandlers: make(NetMessageHandlerMap),

		sessionManager: NewSessionManager(),
	}

	srv.init(opts...)

	return srv
}

func (s *Server) init(opts ...ServerOption) {
	for _, o := range opts {
		o(s)
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
}

func (s *Server) Name() string {
	return KindTcp
}

func (s *Server) Endpoint() (*url.URL, error) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	if err := s.listenAndEndpoint(); err != nil {
		return nil, err
	}
	return s.endpoint, nil
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

		s.endpoint = transport.NewRegistryEndpoint(KindTcp, addr)
	}

	return nil
}

func (s *Server) RegisterMessageHandler(messageType NetMessageType, handler NetMessageHandler, binder Creator) {
	s.handlerMu.Lock()
	defer s.handlerMu.Unlock()

	if _, ok := s.messageHandlers[messageType]; ok {
		return
	}

	s.messageHandlers[messageType] = MessageHandlerData{
		handler, binder,
	}
}

func RegisterServerMessageHandler[T any](srv *Server, messageType NetMessageType, handler func(SessionID, *T) error) {
	srv.RegisterMessageHandler(messageType,
		func(sessionId SessionID, payload NetMessagePayload) error {
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

// GetMessageHandler find message handler
func (s *Server) GetMessageHandler(msgType NetMessageType) (error, *MessageHandlerData) {
	s.handlerMu.RLock()
	defer s.handlerMu.RUnlock()

	handlerData, ok := s.messageHandlers[msgType]
	if !ok {
		errMsg := fmt.Sprintf("[%d] message handler not found", msgType)
		LogError(errMsg)
		return errors.New(errMsg), nil
	}

	return nil, &handlerData
}

// SendRawData send raw data to client
func (s *Server) SendRawData(sessionId SessionID, message []byte) error {
	session := s.sessionManager.getSession(sessionId)
	if session == nil {
		LogError("session not found:", sessionId)
		return errors.New(fmt.Sprintf("session not found: %s", sessionId))
	}

	session.SendMessage(message)

	return nil
}

func (s *Server) BroadcastRawData(message []byte) {
	s.sessionManager.rangeSessions(
		func(id SessionID, session *Session) bool {
			session.SendMessage(message)
			return false
		},
	)
}

func (s *Server) SendMessage(sessionId SessionID, messageType NetMessageType, message NetMessagePayload) error {
	buf, err := s.marshalNetPacket(messageType, message)
	if err != nil {
		LogError("marshal message exception:", err)
		return errors.New(fmt.Sprintf("marshal message exception: %s", err.Error()))
	}

	return s.SendRawData(sessionId, buf)
}

func (s *Server) Broadcast(messageType NetMessageType, message NetMessagePayload) {
	buf, err := s.marshalNetPacket(messageType, message)
	if err != nil {
		LogError(" marshal message exception:", err)
		return
	}

	s.BroadcastRawData(buf)
}

func (s *Server) Start(_ context.Context) error {
	if s.running {
		return nil
	}

	s.running = true

	if s.err = s.listenAndEndpoint(); s.err != nil {
		return s.err
	}

	LogInfof("server listening on: %s", s.lis.Addr().String())

	go s.doAccept()

	return nil
}

func (s *Server) Stop(_ context.Context) error {
	if !s.running {
		return nil
	}

	s.running = false

	LogInfo("server stopping ...")

	var err error

	if s.lis != nil {
		err = s.lis.Close()
		s.lis = nil
	}
	s.err = nil

	LogInfo("server stopped")

	return err
}

func (s *Server) marshalNetPacket(messageType NetMessageType, message NetMessagePayload) ([]byte, error) {
	if s.netPacketMarshaler != nil {
		return s.netPacketMarshaler(messageType, message)
	} else {
		return s.defaultMarshalNetPacket(messageType, message)
	}
}

func (s *Server) defaultMarshalNetPacket(messageType NetMessageType, message NetMessagePayload) ([]byte, error) {
	var err error
	var msg NetPacket
	msg.Type = messageType
	msg.Payload, err = broker.Marshal(s.codec, message)
	if err != nil {
		return nil, err
	}

	buff, err := msg.Marshal()
	if err != nil {
		return nil, err
	}

	return buff, nil
}

func (s *Server) unmarshalNetPacket(buf []byte) (*MessageHandlerData, NetMessagePayload, error) {
	if s.netPacketUnmarshaler != nil {
		return s.netPacketUnmarshaler(buf)
	} else {
		return s.defaultUnmarshalNetPacket(buf)
	}
}

func (s *Server) defaultUnmarshalNetPacket(buf []byte) (handler *MessageHandlerData, payload NetMessagePayload, err error) {
	var msg NetPacket
	if err = msg.Unmarshal(buf); err != nil {
		LogErrorf("decode message exception: %s", err)
		return
	}

	if err, handler = s.GetMessageHandler(msg.Type); err != nil {
		return
	}

	if payload = handler.Create(); payload == nil {
		payload = msg.Payload
	} else {
		if err = broker.Unmarshal(s.codec, msg.Payload, &payload); err != nil {
			LogErrorf("unmarshal message exception: %s", err)
			return
		}
	}

	return
}

func (s *Server) defaultHandleSocketRawData(sessionId SessionID, buf []byte) error {
	var err error
	var handler *MessageHandlerData
	var payload NetMessagePayload

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

// doAccept accept connection handler
func (s *Server) doAccept() {
	for {
		s.stateMu.RLock()
		listener := s.lis
		running := s.running
		s.stateMu.RUnlock()

		if !running || listener == nil {
			return
		}

		conn, err := s.lis.Accept()
		if err != nil {
			s.stateMu.RLock()
			running = s.running
			s.stateMu.RUnlock()

			if !running || errors.Is(err, net.ErrClosed) {
				LogInfof("accept loop stopped: %s", err.Error())
				return
			}

			LogErrorf("accept connection failed: %s", err.Error())
			continue
		}

		session := NewSession(conn, s)
		s.sessionManager.addSession(session, nil)
		session.Listen()
	}
}

// removeSession removes a session from the manager, used by SessionHooks.
func (s *Server) removeSession(session *Session) {
	if s.sessionManager == nil {
		return
	}
	s.sessionManager.removeSession(session, nil)
}

// handleSocketRawData dispatches raw message bytes, used by SessionHooks.
func (s *Server) handleSocketRawData(sessionId SessionID, buf []byte) error {
	if s.socketRawDataHandler != nil {
		return s.socketRawDataHandler(sessionId, buf)
	} else {
		return s.defaultHandleSocketRawData(sessionId, buf)
	}
}
