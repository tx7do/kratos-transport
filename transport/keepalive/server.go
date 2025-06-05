package keepalive

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync/atomic"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	kratosTransport "github.com/go-kratos/kratos/v2/transport"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/tx7do/kratos-transport/transport"
)

var (
	_ kratosTransport.Server     = (*Server)(nil)
	_ kratosTransport.Endpointer = (*Server)(nil)
)

type Server struct {
	*grpc.Server
	health *health.Server

	grpcOpts []grpc.ServerOption

	lis      net.Listener
	endpoint *url.URL

	network string
	address string

	serviceKind string

	started atomic.Bool
	err     error
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		health: health.NewServer(),

		network: "tcp",
		address: "",

		serviceKind: KindKeepAlive,

		started: atomic.Bool{},
	}

	srv.init(opts...)

	return srv
}

func (s *Server) init(opts ...ServerOption) {
	for _, o := range opts {
		o(s)
	}

	if s.address == "" {
		addr, _ := generateHost()
		s.address = fmt.Sprintf("%s:%d", addr, generatePort())
	}

	s.Server = grpc.NewServer(s.grpcOpts...)

	grpc_health_v1.RegisterHealthServer(s.Server, s.health)
}

func (s *Server) Name() string {
	return KindKeepAlive
}

func (s *Server) Start(_ context.Context) error {
	if s.started.Load() {
		return nil
	}

	if s.err = s.listenAndEndpoint(); s.err != nil {
		return s.err
	}

	s.started.Store(true)

	s.health.Resume()

	log.Infof("[%s] server listening on: %s", s.serviceKind, s.lis.Addr().String())

	if s.err = s.Serve(s.lis); !errors.Is(s.err, http.ErrServerClosed) {
		return s.err
	}

	return nil
}

func (s *Server) Stop(_ context.Context) error {
	if !s.started.Load() {
		return nil
	}

	log.Infof("[%s] server stopping...", s.serviceKind)

	s.started.Store(false)

	s.health.Shutdown()
	s.GracefulStop()
	s.err = nil

	log.Infof("[%s] service stopped", s.serviceKind)

	return nil
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

		s.endpoint = transport.NewRegistryEndpoint(s.serviceKind, addr)
	}

	return nil
}

func (s *Server) Endpoint() (*url.URL, error) {
	if err := s.listenAndEndpoint(); err != nil {
		return nil, err
	}
	return s.endpoint, nil
}
