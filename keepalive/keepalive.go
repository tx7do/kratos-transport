package keepalive

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type Service struct {
	*grpc.Server
	health *health.Server

	grpcOpts []grpc.ServerOption

	lis      net.Listener
	endpoint *url.URL
	host     string
}

func NewKeepAliveService(opts ...ServiceOption) *Service {
	srv := &Service{
		health: health.NewServer(),
	}

	srv.init(opts...)

	return srv
}

func (s *Service) init(opts ...ServiceOption) {
	for _, o := range opts {
		o(s)
	}

	s.Server = grpc.NewServer(s.grpcOpts...)
	grpc_health_v1.RegisterHealthServer(s.Server, s.health)
}

func (s *Service) Start() error {
	if err := s.generateEndpoint(s.host); err != nil {
		return err
	}

	s.health.Resume()

	log.Debugf("keep alive service started at %s", s.lis.Addr().String())

	err := s.Serve(s.lis)
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *Service) Stop(_ context.Context) error {
	s.health.Shutdown()
	s.GracefulStop()

	log.Debug("keep alive service stopping")

	return nil
}

func (s *Service) generateEndpoint(host string) error {
	if s.endpoint != nil {
		return nil
	}

	for {
		// generate a port
		port := generatePort(10000, 65535)

		// query network interface
		if host == "" {
			if itf, ok := os.LookupEnv("KRATOS_TRANSPORT_KEEPALIVE_INTERFACE"); ok {
				h, err := getIPAddress(itf)
				if err != nil {
					return err
				}
				host = h
			}
		}

		addr := fmt.Sprintf("%s:%d", host, port)
		lis, err := net.Listen("tcp", addr)
		if err == nil && lis != nil {
			s.lis = lis
			endpoint, _ := url.Parse("tcp://" + addr)
			s.endpoint = endpoint
			return nil
		}
	}
}

func (s *Service) Endpoint() (*url.URL, error) {
	if err := s.generateEndpoint(s.host); err != nil {
		return nil, err
	}
	return s.endpoint, nil
}
