package fasthttp

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"

	"github.com/go-kratos/kratos/v2/errors"
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
	*fasthttp.Server

	tlsConf *tls.Config
	timeout time.Duration
	addr    string

	err error

	filters []FilterFunc
	ms      []middleware.Middleware
	dec     kHttp.DecodeRequestFunc
	enc     kHttp.EncodeResponseFunc
	ene     kHttp.EncodeErrorFunc

	strictSlash bool
	router      *router.Router
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		timeout:     1 * time.Second,
		dec:         kHttp.DefaultRequestDecoder,
		enc:         kHttp.DefaultResponseEncoder,
		ene:         kHttp.DefaultErrorEncoder,
		strictSlash: true,
		router:      router.New(),
	}

	srv.init(opts...)

	return srv
}

func (s *Server) init(opts ...ServerOption) {
	for _, o := range opts {
		o(s)
	}

	s.Server = &fasthttp.Server{
		TLSConfig: s.tlsConf,
		Handler:   FilterChain(s.filters...)(s.router.Handler),
	}

	s.router.RedirectTrailingSlash = s.strictSlash
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
	log.Infof("[fasthttp] server listening on: %s", s.addr)

	var err error
	if s.tlsConf != nil {
		err = s.Server.ListenAndServeTLS(s.addr, "", "")
	} else {
		err = s.Server.ListenAndServe(s.addr)
	}
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *Server) Stop(_ context.Context) error {
	log.Info("[fasthttp] server stopping")
	return s.Server.Shutdown()
}

func (s *Server) Handle(method, path string, handler fasthttp.RequestHandler) {
	s.router.Handle(method, path, handler)
}

func (s *Server) GET(path string, handler fasthttp.RequestHandler) {
	s.Handle(fasthttp.MethodGet, path, handler)
}

func (s *Server) HEAD(path string, handler fasthttp.RequestHandler) {
	s.Handle(fasthttp.MethodHead, path, handler)
}

func (s *Server) POST(path string, handler fasthttp.RequestHandler) {
	s.Handle(fasthttp.MethodPost, path, handler)
}

func (s *Server) PUT(path string, handler fasthttp.RequestHandler) {
	s.Handle(fasthttp.MethodPut, path, handler)
}

func (s *Server) PATCH(path string, handler fasthttp.RequestHandler) {
	s.Handle(fasthttp.MethodPatch, path, handler)
}

func (s *Server) DELETE(path string, handler fasthttp.RequestHandler) {
	s.Handle(fasthttp.MethodDelete, path, handler)
}

func (s *Server) CONNECT(path string, handler fasthttp.RequestHandler) {
	s.Handle(fasthttp.MethodConnect, path, handler)
}

func (s *Server) OPTIONS(path string, handler fasthttp.RequestHandler) {
	s.Handle(fasthttp.MethodOptions, path, handler)
}

func (s *Server) TRACE(path string, handler fasthttp.RequestHandler) {
	s.Handle(fasthttp.MethodTrace, path, handler)
}
