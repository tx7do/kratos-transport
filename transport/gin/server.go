package gin

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"

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
	*gin.Engine
	server *http.Server

	tlsConf *tls.Config
	timeout time.Duration

	network  string
	address  string
	endpoint *url.URL
	lis      net.Listener

	err error

	filters []kHttp.FilterFunc
	ms      []middleware.Middleware
	dec     kHttp.DecodeRequestFunc
	enc     kHttp.EncodeResponseFunc
	ene     kHttp.EncodeErrorFunc
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		network: "tcp",
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
		Addr:      s.address,
		Handler:   s.Engine,
		TLSConfig: s.tlsConf,
	}
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

		s.endpoint = &url.URL{Scheme: KindGin, Host: addr}
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
		err = s.server.ServeTLS(s.lis, "", "")
	} else {
		err = s.server.Serve(s.lis)
	}
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	LogInfo("server stopping...")

	err := s.server.Shutdown(ctx)
	s.err = nil

	LogInfo("server stopped.")

	return err
}

func (s *Server) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	s.Engine.ServeHTTP(res, req)
}
