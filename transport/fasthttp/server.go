package fasthttp

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"

	"github.com/go-kratos/kratos/v2/errors"
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
	*fasthttp.Server

	tlsConf *tls.Config
	timeout time.Duration

	network  string
	address  string
	endpoint *url.URL
	lis      net.Listener

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
		network:     "tcp",
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
	if err := s.listenAndEndpoint(); err != nil {
		return nil, err
	}
	return s.endpoint, nil
}

func (s *Server) listenAndEndpoint() error {
	if s.lis == nil {
		lis, err := net.Listen(s.network, s.address)
		if err != nil {
			return err
		}
		s.lis = lis
	}

	if s.endpoint == nil {
		// 如果传入的是完整的ip地址，则不需要调整。
		// 如果传入的只有端口号，则会调整为完整的地址，但，IP地址或许会不正确。
		addr, err := transport.AdjustAddress(s.address, s.lis)
		if err != nil {
			return err
		}

		s.endpoint = &url.URL{Scheme: KindFastHttp, Host: addr}
	}

	return nil
}

func (s *Server) Start(_ context.Context) error {
	if err := s.listenAndEndpoint(); err != nil {
		return err
	}

	LogInfof("server listening on: %s", s.address)

	var err error
	if s.tlsConf != nil {
		err = s.Server.ServeTLS(s.lis, "", "")
	} else {
		err = s.Server.Serve(s.lis)
	}
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *Server) Stop(_ context.Context) error {
	LogInfo("server stopping...")

	err := s.Server.Shutdown()
	s.err = nil

	LogInfo("server stopped.")

	return err
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
