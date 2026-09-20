package signalr

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/go-kratos/kratos/v2/encoding"
	kratosTransport "github.com/go-kratos/kratos/v2/transport"

	"github.com/philippseith/signalr"

	"github.com/tx7do/kratos-transport/transport"
)

var (
	_ kratosTransport.Server     = (*Server)(nil)
	_ kratosTransport.Endpointer = (*Server)(nil)
)

type Server struct {
	signalr.Server

	lis      net.Listener
	tlsConf  *tls.Config
	endpoint *url.URL

	network string
	address string

	keepAliveInterval  time.Duration
	chanReceiveTimeout time.Duration

	streamBufferCapacity uint

	debug bool

	err   error
	codec encoding.Codec

	allowedOrigins   []string
	allowCredentials bool

	hub signalr.HubInterface

	router  *http.ServeMux
	running atomic.Bool
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		network:              "tcp",
		address:              ":0",
		router:               http.NewServeMux(),
		keepAliveInterval:    2 * time.Second,
		chanReceiveTimeout:   200 * time.Millisecond,
		streamBufferCapacity: 5,
		debug:                false,
	}

	srv.init(opts...)

	return srv
}

func (s *Server) Name() string {
	return KindSignalR
}

func (s *Server) Start(_ context.Context) error {
	if s.running.Load() {
		// 已在监听：避免同一 listener 叠两个 accept 循环
		return nil
	}

	// Endpoint() 可能已预创建 lis，Start 前先释放以获得干净的监听
	if s.lis != nil {
		_ = s.lis.Close()
		s.lis = nil
		s.endpoint = nil
	}

	if s.err = s.listenAndEndpoint(); s.err != nil {
		return s.err
	}

	if s.err != nil {
		return s.err
	}

	LogInfof("server listening on: %s", s.lis.Addr().String())

	//handler := handlers.CORS(
	//	handlers.AllowedMethods([]string{"GET", "POST", "OPTIONS"}),
	//	handlers.AllowedOrigins([]string{"http://localhost:63342"}),
	//	handlers.AllowedHeaders([]string{"x-requested-with", "x-signalr-user-agent"}),
	//	handlers.ExposedHeaders([]string{"x-requested-with", "x-signalr-user-agent"}),
	//	handlers.AllowCredentials(),
	//)(s.router)

	handler := s.CORS(s.router)

	s.running.Store(true)

	var err error
	if s.tlsConf != nil {
		err = http.ServeTLS(s.lis, handler, "", "")
	} else {
		err = http.Serve(s.lis, handler)
	}
	s.running.Store(false)
	// Stop 关闭 listener 后 Serve 返回 use of closed（非 ErrServerClosed），属正常停止
	if err != nil && !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, net.ErrClosed) {
		return err
	}

	return nil
}

func (s *Server) Stop(_ context.Context) error {
	LogInfo("server stopping...")

	s.running.Store(false)

	var err error
	if s.lis != nil {
		err = s.lis.Close()
	}
	// 释放 listener 与 endpoint：下一次 Start 通过 listenAndEndpoint 重新监听（重启支持）
	s.lis = nil
	s.endpoint = nil
	s.err = nil

	LogInfo("server stopped.")

	return err
}

func (s *Server) Endpoint() (*url.URL, error) {
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

		s.endpoint = transport.NewRegistryEndpoint(KindSignalR, addr)
	}

	return nil
}

func (s *Server) MapHTTP(path string) {
	s.Server.MapHTTP(signalr.WithHTTPServeMux(s.router), path)
}

func (s *Server) init(opts ...ServerOption) {
	for _, o := range opts {
		o(s)
	}

	options := []func(signalr.Party) error{
		signalr.Logger(&logger{}, s.debug),
		//signalr.HubFactory(s.createHub),
		signalr.KeepAliveInterval(s.keepAliveInterval),
		signalr.ChanReceiveTimeout(s.chanReceiveTimeout),
		signalr.StreamBufferCapacity(s.streamBufferCapacity),
	}

	// 未传 WithHub 时 SimpleHubFactory(nil) 会在库内反射解引用直接 panic
	if s.hub != nil {
		options = append(options, signalr.SimpleHubFactory(s.hub))
	}

	server, err := signalr.NewServer(context.Background(),
		options...,
	)
	if err != nil {
		s.err = err
		return
	}
	s.Server = server
}
