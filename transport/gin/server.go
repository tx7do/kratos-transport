package gin

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	kHttp "github.com/go-kratos/kratos/v2/transport/http"
)

var (
	_ transport.Server     = (*Server)(nil)
	_ transport.Endpointer = (*Server)(nil)
)

type Server struct {
	*gin.Engine
	server *http.Server

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
	s.Engine = gin.New()

	for _, o := range opts {
		o(s)
	}

	s.server = &http.Server{
		Addr:      s.addr,
		Handler:   s.Engine,
		TLSConfig: s.tlsConf,
	}
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
	log.Infof("[GIN] server listening on: %s", s.addr)

	var err error
	if s.tlsConf != nil {
		err = s.server.ListenAndServeTLS("", "")
	} else {
		err = s.server.ListenAndServe()
	}
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	log.Info("[GIN] server stopping")
	return s.server.Shutdown(ctx)
}

func (s *Server) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	s.Engine.ServeHTTP(res, req)
}
