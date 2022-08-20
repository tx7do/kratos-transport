package graphql

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"

	"github.com/gorilla/mux"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
)

var (
	_ transport.Server     = (*Server)(nil)
	_ transport.Endpointer = (*Server)(nil)
)

type Server struct {
	*http.Server
	es graphql.ExecutableSchema

	lis      net.Listener
	tlsConf  *tls.Config
	endpoint *url.URL
	router   *mux.Router

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

	srv.err = srv.listenAndEndpoint()

	return srv
}

func (s *Server) Name() string {
	return "graphql"
}

func (s *Server) Handle(path string, es graphql.ExecutableSchema) {
	s.router.Handle(path, handler.NewDefaultServer(es))
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

	addr := s.address

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

	s.endpoint, s.err = url.Parse(addr)

	return nil
}

func (s *Server) Endpoint() (*url.URL, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.endpoint, nil
}

func (s *Server) Start(ctx context.Context) error {
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
	log.Info("[graphql] server stopping")
	return s.Shutdown(ctx)
}
