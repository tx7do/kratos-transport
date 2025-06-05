package iris

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/kataras/iris/v12"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	kratosTransport "github.com/go-kratos/kratos/v2/transport"

	"github.com/tx7do/kratos-transport/transport"
)

var (
	_ kratosTransport.Server     = (*Server)(nil)
	_ kratosTransport.Endpointer = (*Server)(nil)
)

type Server struct {
	*iris.Application

	endpoint *url.URL
	timeout  time.Duration
	addr     string

	certFile, keyFile string

	err error
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		timeout:     1 * time.Second,
		Application: iris.New(),
	}

	srv.init(opts...)

	return srv
}

func (s *Server) init(opts ...ServerOption) {
	for _, o := range opts {
		o(s)
	}
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
		s.endpoint = &url.URL{Scheme: KindIris, Host: addr}
	}

	return nil
}

func (s *Server) Name() string {
	return KindIris
}

func (s *Server) Start(_ context.Context) error {
	if s.err = s.listenAndEndpoint(); s.err != nil {
		return s.err
	}

	log.Infof("[Iris] server listening on: %s", s.addr)

	var err error
	if len(s.certFile) != 0 && len(s.keyFile) != 0 {
		err = s.Application.Run(iris.TLS(s.addr, s.certFile, s.keyFile))
	} else {
		err = s.Application.Listen(s.addr)
	}
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	log.Info("[Iris] server stopping...")

	err := s.Application.Shutdown(ctx)
	s.err = nil

	log.Info("[Iris] server stopped.")

	return err
}

func (s *Server) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	s.Application.ServeHTTP(res, req)
}
