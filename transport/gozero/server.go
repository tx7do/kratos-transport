package gozero

import (
	"context"
	"fmt"
	"net"
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
	started  bool
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
		host := s.cfg.Host
		if host == "" || host == "0.0.0.0" {
			ip, _ := transport.GetLocalIP()
			host = ip
		}
		addr := host + ":" + fmt.Sprint(s.cfg.Port)
		s.endpoint = transport.NewRegistryEndpoint(KindGoZero, addr)
	}
	return nil
}

func (s *Server) Start(_ context.Context) error {
	if s.started {
		return nil
	}

	if err := s.listenAndEndpoint(); err != nil {
		return err
	}

	LogInfof("server listening on: %d", s.cfg.Port)

	// go-zero 的 rest.Server 对监听冲突等错误会直接 panic，
	// 提前占位检测以给出可读的错误而非崩溃
	probe, probeErr := net.Listen("tcp", fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port))
	if probeErr != nil {
		return probeErr
	}
	_ = probe.Close()

	s.started = true
	s.Server.Start()

	return nil
}

func (s *Server) Stop(_ context.Context) error {
	LogInfo("server stopping...")

	s.started = false
	s.Server.Stop()
	s.err = nil

	LogInfo("server stopped")

	return nil
}

func (s *Server) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {
}
