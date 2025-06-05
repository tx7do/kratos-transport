package iris

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kataras/iris/v12"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	kratosTransport "github.com/go-kratos/kratos/v2/transport"
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

	s.endpoint, _ = url.Parse(s.addr)
}

func (s *Server) Endpoint() (*url.URL, error) {
	addr := s.addr

	prefix := "http://"
	if len(s.certFile) == 0 && len(s.keyFile) == 0 {
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

func (s *Server) Name() string {
	return string(KindIris)
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
