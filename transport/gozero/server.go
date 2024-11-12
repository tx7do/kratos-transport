package gozero

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-kratos/kratos/v2/log"
	kratosTransport "github.com/go-kratos/kratos/v2/transport"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/rest"
)

var (
	_ kratosTransport.Server     = (*Server)(nil)
	_ kratosTransport.Endpointer = (*Server)(nil)
)

type Server struct {
	*rest.Server

	cfg rest.RestConf

	err error
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
	prefix := "http://"
	if len(s.cfg.CertFile) == 0 && len(s.cfg.KeyFile) == 0 {
		prefix = "http://"
	} else {
		prefix = "https://"
	}

	var endpoint *url.URL
	endpoint, s.err = url.Parse(fmt.Sprint(prefix, ":", s.cfg.Port))

	return endpoint, s.err
}

func (s *Server) Start(ctx context.Context) error {
	log.Infof("[go-zero] server listening on: %d", s.cfg.Port)

	s.Server.Start()

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	log.Info("[go-zero] server stopping")
	s.Server.Stop()
	return nil
}

func (s *Server) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {
}
