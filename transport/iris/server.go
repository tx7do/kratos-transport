package iris

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/kataras/iris/v12"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
)

var (
	_ transport.Server     = (*Server)(nil)
	_ transport.Endpointer = (*Server)(nil)
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

	s.endpoint, _ = url.Parse(s.addr)
}

func (s *Server) Endpoint() (*url.URL, error) {
	return s.endpoint, nil
}

func (s *Server) Start(ctx context.Context) error {
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
	log.Info("[Iris] server stopping")
	return s.Application.Shutdown(ctx)
}

func (s *Server) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	s.Application.ServeHTTP(res, req)
}
