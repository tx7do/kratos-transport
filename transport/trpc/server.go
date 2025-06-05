package trpc

import (
	"context"
	"fmt"
	"net"
	"net/url"

	trpcGo "trpc.group/trpc-go/trpc-go"
	trpcServer "trpc.group/trpc-go/trpc-go/server"

	kratosTransport "github.com/go-kratos/kratos/v2/transport"

	"github.com/tx7do/kratos-transport/transport"
)

var (
	_ kratosTransport.Server     = (*Server)(nil)
	_ kratosTransport.Endpointer = (*Server)(nil)
)

type Server struct {
	Server *trpcServer.Server

	trpcOptions []trpcServer.Option
	address     string

	err error

	endpoint *url.URL
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{}

	srv.init(opts...)

	return srv
}

func (s *Server) init(opts ...ServerOption) {
	for _, o := range opts {
		o(s)
	}
}

func (s *Server) Name() string {
	return KindTRPC
}

func (s *Server) Endpoint() (*url.URL, error) {
	if err := s.listenAndEndpoint(); err != nil {
		return nil, err
	}
	return s.endpoint, nil
}

func (s *Server) listenAndEndpoint() error {
	if s.endpoint == nil {

		host, port, err := net.SplitHostPort(s.address)
		if err != nil {
			return err
		}

		if host == "" {
			ip, _ := transport.GetLocalIP()
			host = ip
		}

		addr := host + ":" + fmt.Sprint(port)
		s.endpoint = transport.NewRegistryEndpoint(KindTRPC, addr)
	}

	return nil
}

func (s *Server) Start(_ context.Context) error {
	if s.err = s.listenAndEndpoint(); s.err != nil {
		return s.err
	}

	s.Server = trpcGo.NewServer(s.trpcOptions...)
	if s.Server == nil {
		return fmt.Errorf("failed to create trpc server")
	}

	if s.err = s.Server.Serve(); s.err != nil {
		LogErrorf("server serve error: %v", s.err)
		return s.err
	}

	return nil
}

func (s *Server) Stop(_ context.Context) error {
	LogInfo("server stopping...")

	err := s.Server.Close(nil)

	LogInfo("server stopped.")

	return err
}

func (s *Server) AddService(serviceName string, service trpcServer.Service) {
	if s.Server == nil {
		LogErrorf("server is not initialized, cannot add service %s", serviceName)
		return
	}

	s.Server.AddService(serviceName, service)
}
