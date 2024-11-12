package hertz

import (
	"context"
	"crypto/tls"
	"net/url"
	"strings"
	"time"

	hertz "github.com/cloudwego/hertz/pkg/app/server"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	kratosTransport "github.com/go-kratos/kratos/v2/transport"
	kHttp "github.com/go-kratos/kratos/v2/transport/http"
)

var (
	_ kratosTransport.Server     = (*Server)(nil)
	_ kratosTransport.Endpointer = (*Server)(nil)
)

type Server struct {
	*hertz.Hertz

	tlsConf *tls.Config
	timeout time.Duration
	addr    string

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
	addr := s.addr

	prefix := "http://"
	if s.tlsConf == nil {
		if !strings.HasPrefix(addr, "http://") {
			prefix = "http://"
		}
	} else {
		if !strings.HasPrefix(addr, "https://") {
			prefix = "https://"
		}
	}
	addr = prefix + addr

	var endpoint *url.URL
	endpoint, s.err = url.Parse(addr)

	return endpoint, s.err
}

func (s *Server) Start(ctx context.Context) error {
	log.Infof("[hertz] server listening on: %s", s.addr)

	return s.Hertz.Run()
}

func (s *Server) Stop(ctx context.Context) error {
	log.Info("[hertz] server stopping")
	return s.Hertz.Shutdown(ctx)
}
