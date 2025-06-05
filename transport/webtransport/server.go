package webtransport

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/quic-go/quic-go"

	"github.com/go-kratos/kratos/v2/encoding"
	kratosTransport "github.com/go-kratos/kratos/v2/transport"
	"github.com/tx7do/kratos-transport/broker"

	"github.com/quic-go/quic-go/http3"

	"github.com/tx7do/kratos-transport/transport"
)

const (
	KindWebtransport = "webtransport"
)

var (
	_ kratosTransport.Server     = (*Server)(nil)
	_ kratosTransport.Endpointer = (*Server)(nil)
)

type Server struct {
	*http3.Server

	tlsConf  *tls.Config
	endpoint *url.URL
	timeout  time.Duration

	mux         *http.ServeMux
	path        string
	strictSlash bool

	ctx       context.Context // is closed when Close is called
	ctxCancel context.CancelFunc
	refCount  sync.WaitGroup

	messageHandlers MessageHandlerMap
	connectHandler  ConnectHandler
	codec           encoding.Codec
}

func NewServer(opts ...ServerOption) *Server {
	ctx, ctxCancel := context.WithCancel(context.Background())
	srv := &Server{
		ctx:       ctx,
		ctxCancel: ctxCancel,
		mux:       http.NewServeMux(),

		messageHandlers: make(MessageHandlerMap),
		codec:           encoding.GetCodec("json"),
	}
	srv.init(opts...)
	return srv
}

func (s *Server) init(opts ...ServerOption) {
	const idleTimeout = 30 * time.Second

	s.Server = &http3.Server{
		Addr: ":443",
		QUICConfig: &quic.Config{
			MaxIdleTimeout:  idleTimeout,
			KeepAlivePeriod: idleTimeout / 2,
		},
	}

	for _, o := range opts {
		o(s)
	}

	if s.tlsConf == nil {
		s.tlsConf = generateTLSConfig(alpnQuicTransport)
	}
	s.Server.TLSConfig = s.tlsConf

	timeout := s.timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	if s.Server.AdditionalSettings == nil {
		s.Server.AdditionalSettings = make(map[uint64]uint64)
	}

	s.mux.HandleFunc(s.path, s.addHandler)
	s.Server.Handler = s.mux
}

func (s *Server) Endpoint() (*url.URL, error) {
	if err := s.listenAndEndpoint(); err != nil {
		return nil, err
	}
	return s.endpoint, nil
}

func (s *Server) listenAndEndpoint() error {
	if s.endpoint == nil {
		host, port, err := net.SplitHostPort(s.Addr)
		if err != nil {
			return err
		}

		if host == "" {
			ip, _ := transport.GetLocalIP()
			host = ip
		}

		addr := host + ":" + fmt.Sprint(port)
		s.endpoint = &url.URL{Scheme: KindWebtransport, Host: addr}
	}

	return nil
}

func (s *Server) Start(_ context.Context) error {
	if err := s.listenAndEndpoint(); err != nil {
		return err
	}

	LogInfof("server listening on: %s", s.Addr)

	if err := s.ListenAndServe(); err != nil {
		LogErrorf("start server failed: %s", err.Error())
		return err
	}

	return nil
}

func (s *Server) Stop(_ context.Context) error {
	LogInfo("server stopping...")

	if s.ctxCancel != nil {
		s.ctxCancel()
	}

	err := s.Server.Close()
	s.refCount.Wait()

	LogInfo("server stopped.")

	return err
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

func (s *Server) messageHandler(sessionId SessionID, buf []byte) error {
	var msg Message
	if err := msg.Unmarshal(buf); err != nil {
		LogErrorf("decode message exception: %s", err)
		return err
	}

	handlerData, ok := s.messageHandlers[msg.Type]
	if !ok {
		LogError("message type not found:", msg.Type)
		return errors.New("message handler not found")
	}

	var payload MessagePayload

	if handlerData.Binder != nil {
		payload = handlerData.Binder()
	} else {
		payload = msg.Body
	}

	if err := broker.Unmarshal(s.codec, msg.Body, &payload); err != nil {
		LogErrorf("unmarshal message exception: %s", err)
		return err
	}

	if err := handlerData.Handler(sessionId, payload); err != nil {
		LogErrorf("message handler exception: %s", err)
		return err
	}

	return nil
}

func (s *Server) addHandler(w http.ResponseWriter, r *http.Request) {
}
