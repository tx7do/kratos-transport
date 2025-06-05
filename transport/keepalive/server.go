package keepalive

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"

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
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		health: health.NewServer(),

		network: "tcp",
		address: "",
	}

	srv.init(opts...)

	return srv
}

func (s *Server) init(opts ...ServerOption) {
	for _, o := range opts {
		o(s)
	}

	if s.address == "" {
		s.address = fmt.Sprintf(":%d", generatePort())
	}

	s.Server = grpc.NewServer(s.grpcOpts...)

	grpc_health_v1.RegisterHealthServer(s.Server, s.health)
}

func (s *Server) Name() string {
	return KindKeepAlive
}

func (s *Server) Start(_ context.Context) error {
	if err := s.listenAndEndpoint(); err != nil {
		return err
	}

	s.health.Resume()

	log.Debugf("keep alive server started at %s", s.lis.Addr().String())

	if err := s.Serve(s.lis); !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *Server) Stop(_ context.Context) error {
	log.Debug("keep alive service stopping...")

	s.health.Shutdown()
	s.GracefulStop()

	log.Debug("keep alive service stopped")

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

		s.endpoint = &url.URL{Scheme: KindKeepAlive, Host: addr}
	}

	return nil
}

func (s *Server) Endpoint() (*url.URL, error) {
	if err := s.listenAndEndpoint(); err != nil {
		return nil, err
	}
	return s.endpoint, nil
}
