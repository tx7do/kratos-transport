package gozero

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	kratosTransport "github.com/go-kratos/kratos/v2/transport"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/rest"

	"github.com/tx7do/kratos-transport/transport"
)

var (
	_ kratosTransport.Server     = (*Server)(nil)
	_ kratosTransport.Endpointer = (*Server)(nil)
)

type Server struct {
	*rest.Server

	cfg rest.RestConf

	err error

	endpoint *url.URL
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{}

	srv.init(opts...)

	return srv
}

func (s *Server) init(opts ...ServerOption) {
	s.cfg.MaxConns = 500

	s.cfg.ServiceConf = service.ServiceConf{
		Log: logx.LogConf{
			Mode: "console",
		},
	}

	for _, o := range opts {
		o(s)
	}

	s.Server = rest.MustNewServer(s.cfg)
}

func (s *Server) Endpoint() (*url.URL, error) {
	if err := s.listenAndEndpoint(); err != nil {
		return nil, err
	}
	return s.endpoint, nil
}

func (s *Server) listenAndEndpoint() error {
	if s.endpoint == nil {
		ip, _ := transport.GetLocalIP()
		host := ip + ":" + fmt.Sprint(s.cfg.Port)
		s.endpoint = transport.NewRegistryEndpoint(KindGoZero, host)
	}
	return nil
}

func (s *Server) Start(_ context.Context) error {
	if err := s.listenAndEndpoint(); err != nil {
		return err
	}

	LogInfof("server listening on: %d", s.cfg.Port)

	s.Server.Start()

	return nil
}

func (s *Server) Stop(_ context.Context) error {
	LogInfo("server stopping...")

	s.Server.Stop()
	s.err = nil

	LogInfo("server stopped")

	return nil
}

func (s *Server) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {
}
