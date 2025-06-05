package hertz

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"time"

	hertz "github.com/cloudwego/hertz/pkg/app/server"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	kratosTransport "github.com/go-kratos/kratos/v2/transport"
	kHttp "github.com/go-kratos/kratos/v2/transport/http"

	"github.com/tx7do/kratos-transport/transport"
)

var (
	_ kratosTransport.Server     = (*Server)(nil)
	_ kratosTransport.Endpointer = (*Server)(nil)
)

type Server struct {
	*hertz.Hertz

	tlsConf  *tls.Config
	timeout  time.Duration
	addr     string
	endpoint *url.URL

	err error

	filters []kHttp.FilterFunc
	ms      []middleware.Middleware
	dec     kHttp.DecodeRequestFunc
	enc     kHttp.EncodeResponseFunc
	ene     kHttp.EncodeErrorFunc
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		timeout: 1 * time.Second,
		dec:     kHttp.DefaultRequestDecoder,
		enc:     kHttp.DefaultResponseEncoder,
		ene:     kHttp.DefaultErrorEncoder,
	}

	srv.init(opts...)

	return srv
}

func (s *Server) init(opts ...ServerOption) {
	for _, o := range opts {
		o(s)
	}

	s.Hertz = hertz.Default(hertz.WithHostPorts(s.addr), hertz.WithTLS(s.tlsConf))
}

func (s *Server) Endpoint() (*url.URL, error) {
	if err := s.listenAndEndpoint(); err != nil {
		return nil, err
	}
	return s.endpoint, nil
}

func (s *Server) listenAndEndpoint() error {
	if s.endpoint == nil {
		host, port, err := net.SplitHostPort(s.addr)
		if err != nil {
			return err
		}

		if host == "" {
			ip, _ := transport.GetLocalIP()
			host = ip
		}

		addr := host + ":" + fmt.Sprint(port)
		s.endpoint = &url.URL{Scheme: KindHertz, Host: addr}
	}

	return nil
}

func (s *Server) Start(_ context.Context) error {
	if err := s.listenAndEndpoint(); err != nil {
		return err
	}

	log.Infof("[hertz] server listening on: %s", s.addr)

	return s.Hertz.Run()
}

func (s *Server) Stop(ctx context.Context) error {
	log.Info("[hertz] server stopping...")

	err := s.Hertz.Shutdown(ctx)
	s.err = nil

	log.Info("[hertz] server stopped.")

	return err
}
