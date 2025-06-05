package graphql

import (
	"context"
	"crypto/tls"
	"github.com/tx7do/kratos-transport/transport"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"

	"github.com/gorilla/mux"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	kratosTransport "github.com/go-kratos/kratos/v2/transport"
)

var (
	_ kratosTransport.Server     = (*Server)(nil)
	_ kratosTransport.Endpointer = (*Server)(nil)
)

type Server struct {
	*http.Server

	es graphql.ExecutableSchema

	lis      net.Listener
	tlsConf  *tls.Config
	router   *mux.Router
	endpoint *url.URL

	network string
	address string

	strictSlash bool
	timeout     time.Duration

	err error
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		network:     "tcp",
		address:     ":0",
		timeout:     1 * time.Second,
		strictSlash: true,
	}

	srv.init(opts...)

	return srv
}

func (s *Server) Name() string {
	return KindGraphQL
}

func (s *Server) Handle(path string, es graphql.ExecutableSchema) {
	s.router.Handle(path, handler.New(es))
}

func (s *Server) init(opts ...ServerOption) {
	for _, o := range opts {
		o(s)
	}

	s.router = mux.NewRouter().StrictSlash(s.strictSlash)
	s.router.NotFoundHandler = http.DefaultServeMux
	s.router.MethodNotAllowedHandler = http.DefaultServeMux
	//srv.router.Use(srv.filter())

	s.Server = &http.Server{
		TLSConfig: s.tlsConf,
		Handler:   s.router,
	}
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

		s.endpoint = &url.URL{Scheme: KindGraphQL, Host: addr}
	}

	return nil
}

func (s *Server) Endpoint() (*url.URL, error) {
	if err := s.listenAndEndpoint(); err != nil {
		return nil, err
	}
	return s.endpoint, nil
}

func (s *Server) Start(ctx context.Context) error {
	if s.err = s.listenAndEndpoint(); s.err != nil {
		return s.err
	}

	if s.err != nil {
		return s.err
	}

	s.BaseContext = func(net.Listener) context.Context {
		return ctx
	}
	log.Infof("server listening on: %s", s.lis.Addr().String())
	var err error
	if s.tlsConf != nil {
		err = s.ServeTLS(s.lis, "", "")
	} else {
		err = s.Serve(s.lis)
	}
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	log.Info("[graphql] server stopping...")

	err := s.Shutdown(ctx)
	s.err = nil

	log.Info("[graphql] server stopped.")

	return err
}
