package socketio

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/go-kratos/kratos/v2/log"
	kratosTransport "github.com/go-kratos/kratos/v2/transport"

	socketIo "github.com/googollee/go-socket.io"
	"github.com/googollee/go-socket.io/engineio"
	"github.com/googollee/go-socket.io/engineio/transport"
	"github.com/googollee/go-socket.io/engineio/transport/polling"
	"github.com/googollee/go-socket.io/engineio/transport/websocket"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

var (
	_ kratosTransport.Server     = (*Server)(nil)
	_ kratosTransport.Endpointer = (*Server)(nil)
)

type Server struct {
	*socketIo.Server

	lis     net.Listener
	tlsConf *tls.Config

	network string
	address string
	path    string

	err   error
	codec encoding.Codec

	router *mux.Router
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		network: "tcp",
		address: ":0",
		router:  mux.NewRouter(),
		path:    "/socket.io/",
	}

	srv.init(opts...)

	srv.err = srv.listen()

	return srv
}

func (s *Server) Name() string {
	return string(KindSocketIo)
}

func (s *Server) Start(_ context.Context) error {
	if s.err != nil {
		return s.err
	}

	log.Infof("[socket.io] server listening on: %s", s.address)

	go func() {
		if err := s.Server.Serve(); err != nil {
			log.Fatalf("socketio listen error: %s\n", err)
		}
	}()

	handler := handlers.CORS()(s.router)

	if s.tlsConf != nil {
		s.err = http.ServeTLS(s.lis, handler, "", "")
	} else {
		s.err = http.Serve(s.lis, handler)
	}
	if !errors.Is(s.err, http.ErrServerClosed) {
		return s.err
	}

	return nil
}

func (s *Server) Stop(_ context.Context) error {
	log.Info("[socket.io] server stopping")
	_ = s.lis.Close()
	return s.Server.Close()
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

func (s *Server) RegisterConnectHandler(namespace string, f func(socketIo.Conn) error) {
	s.Server.OnConnect(namespace, f)
}

func (s *Server) RegisterDisconnectHandler(namespace string, f func(socketIo.Conn, string)) {
	s.Server.OnDisconnect(namespace, f)
}

func (s *Server) RegisterErrorHandler(namespace string, f func(socketIo.Conn, error)) {
	s.Server.OnError(namespace, f)
}

func (s *Server) RegisterEventHandler(namespace, event string, f interface{}) {
	s.Server.OnEvent(namespace, event, f)
}

func (s *Server) init(opts ...ServerOption) {
	server := socketIo.NewServer(&engineio.Options{
		Transports: []transport.Transport{
			&polling.Transport{
				CheckOrigin: func(r *http.Request) bool { return true },
			},
			&websocket.Transport{
				CheckOrigin: func(r *http.Request) bool { return true },
			},
		},
	})
	if server == nil {
		s.err = errors.New("create socket.io server failed")
		return
	}
	s.Server = server

	for _, o := range opts {
		o(s)
	}

	s.router.Use(mux.CORSMethodMiddleware(s.router))

	s.router.Handle(s.path, server)
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
