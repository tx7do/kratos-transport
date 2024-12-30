package trpc

import (
	"context"
	"net/url"
	trpcGo "trpc.group/trpc-go/trpc-go"
	trpcServer "trpc.group/trpc-go/trpc-go/server"

	kratosTransport "github.com/go-kratos/kratos/v2/transport"
)

var (
	_ kratosTransport.Server     = (*Server)(nil)
	_ kratosTransport.Endpointer = (*Server)(nil)
)

type Server struct {
	Server *trpcServer.Server

	err error
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		Server: trpcGo.NewServer(),
	}

	srv.init(opts...)

	return srv
}

func (s *Server) Name() string {
	return string(KindTRPC)
}

func (s *Server) Endpoint() (*url.URL, error) {
	return nil, nil
}

func (s *Server) init(opts ...ServerOption) {
	for _, o := range opts {
		o(s)
	}
}

func (s *Server) listen() error {

	return nil
}

func (s *Server) Start(_ context.Context) error {
	if err := s.Server.Serve(); err != nil {
		LogErrorf("server serve error: %v", err)
		return err
	}

	return nil
}

func (s *Server) Stop(_ context.Context) error {
	LogInfo("server stopping")

	return s.Server.Close(nil)
}
