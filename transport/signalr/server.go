package signalr

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/go-kratos/kratos/v2/log"
	kratosTransport "github.com/go-kratos/kratos/v2/transport"

	"github.com/philippseith/signalr"
)

var (
	_ kratosTransport.Server     = (*Server)(nil)
	_ kratosTransport.Endpointer = (*Server)(nil)
)

type Server struct {
	signalr.Server

	lis     net.Listener
	tlsConf *tls.Config

	network string
	address string

	keepAliveInterval  time.Duration
	chanReceiveTimeout time.Duration

	streamBufferCapacity uint

	debug bool

	err   error
	codec encoding.Codec

	hub signalr.HubInterface

	router *http.ServeMux
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		network:              "tcp",
		address:              ":0",
		router:               http.NewServeMux(),
		keepAliveInterval:    2 * time.Second,
		chanReceiveTimeout:   200 * time.Millisecond,
		streamBufferCapacity: 5,
		debug:                false,
	}

	srv.init(opts...)

	srv.err = srv.listen()

	return srv
}

func (s *Server) Name() string {
	return string(KindSignalR)
}

func (s *Server) Start(ctx context.Context) error {
	if s.err != nil {
		return s.err
	}

	log.Infof("[signalr] server listening on: %s", s.lis.Addr().String())

	//handler := handlers.CORS(
	//	handlers.AllowedMethods([]string{"GET", "POST", "OPTIONS"}),
	//	handlers.AllowedOrigins([]string{"http://localhost:63342"}),
	//	handlers.AllowedHeaders([]string{"x-requested-with", "x-signalr-user-agent"}),
	//	handlers.ExposedHeaders([]string{"x-requested-with", "x-signalr-user-agent"}),
	//	handlers.AllowCredentials(),
	//)(s.router)

	handler := s.CORS(s.router)

	var err error
	if s.tlsConf != nil {
		err = http.ServeTLS(s.lis, handler, "", "")
	} else {
		err = http.Serve(s.lis, handler)
	}
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	log.Info("[signalr] server stopping")
	return s.lis.Close()
}

func (s *Server) Endpoint() (*url.URL, error) {
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

	var endpoint *url.URL
	endpoint, s.err = url.Parse(addr)
	return endpoint, nil
}

func (s *Server) MapHTTP(path string) {
	s.Server.MapHTTP(signalr.WithHTTPServeMux(s.router), path)
}

func (s *Server) init(opts ...ServerOption) {
	for _, o := range opts {
		o(s)
	}

	server, err := signalr.NewServer(context.Background(),
		signalr.Logger(&logger{}, s.debug),
		//signalr.HubFactory(s.createHub),
		signalr.SimpleHubFactory(s.hub),
		signalr.KeepAliveInterval(s.keepAliveInterval),
		signalr.ChanReceiveTimeout(s.chanReceiveTimeout),
		signalr.StreamBufferCapacity(s.streamBufferCapacity),
	)
	if err != nil {
		s.err = err
		return
	}
	s.Server = server
}

func (s *Server) listen() error {
	if s.lis == nil {
		lis, err := net.Listen(s.network, s.address)
		if err != nil {
			s.err = err
			return err
		}
		s.lis = lis
	}

	return nil
}
