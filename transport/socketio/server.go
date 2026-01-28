package socketio

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"

	"github.com/go-kratos/kratos/v2/encoding"
	kratosTransport "github.com/go-kratos/kratos/v2/transport"

	socketIo "github.com/googollee/go-socket.io"
	"github.com/googollee/go-socket.io/engineio"
	socketIoTransport "github.com/googollee/go-socket.io/engineio/transport"
	"github.com/googollee/go-socket.io/engineio/transport/polling"
	"github.com/googollee/go-socket.io/engineio/transport/websocket"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/tx7do/kratos-transport/transport"
)

var (
	_ kratosTransport.Server     = (*Server)(nil)
	_ kratosTransport.Endpointer = (*Server)(nil)
)

type Server struct {
	*socketIo.Server

	lis      net.Listener
	tlsConf  *tls.Config
	endpoint *url.URL

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

	return srv
}

func (s *Server) Name() string {
	return KindSocketIo
}

func (s *Server) Start(_ context.Context) error {
	if s.err = s.listenAndEndpoint(); s.err != nil {
		return s.err
	}

	if s.err != nil {
		return s.err
	}

	LogInfof("server listening on: %s", s.address)

	go func() {
		if err := s.Server.Serve(); err != nil {
			LogFatalf("socketio listen error: %s\n", err)
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
	LogInfo("server stopping...")

	//_ = s.lis.Close()
	err := s.Server.Close()
	s.err = nil

	LogInfo("server stopped")

	return err
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

		s.endpoint = transport.NewRegistryEndpoint(KindSocketIo, addr)
	}

	return nil
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

func (s *Server) RegisterEventHandler(namespace, event string, f any) {
	s.Server.OnEvent(namespace, event, f)
}

func (s *Server) init(opts ...ServerOption) {
	server := socketIo.NewServer(&engineio.Options{
		Transports: []socketIoTransport.Transport{
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
