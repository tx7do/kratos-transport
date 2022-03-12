package websocket

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
	ws "github.com/gorilla/websocket"
)

type Message struct {
	Body []byte
}

type Handler func(string, *Message) error
type EchoHandler func(string, *Message) (*Message, error)
type ConnectHandler func(string, bool)

var (
	_ transport.Server     = (*Server)(nil)
	_ transport.Endpointer = (*Server)(nil)
)

type Server struct {
	*http.Server
	lis         net.Listener
	tlsConf     *tls.Config
	endpoint    *url.URL
	strictSlash bool

	err error

	network string
	address string
	timeout time.Duration

	log *log.Helper

	readHandler    EchoHandler
	connectHandler ConnectHandler
	path           string

	clients  ClientMap
	upgrader *ws.Upgrader

	register   chan *Client
	unregister chan *Client
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		network:     "tcp",
		address:     ":0",
		timeout:     1 * time.Second,
		strictSlash: true,
		log:         log.NewHelper(log.GetLogger()),

		clients: ClientMap{},
		upgrader: &ws.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			}},

		register:   make(chan *Client),
		unregister: make(chan *Client),
	}

	srv.init(opts...)

	srv.err = srv.listen()

	return srv
}

func (s *Server) String() string {
	return "websocket"
}

func (s *Server) init(opts ...ServerOption) {
	for _, o := range opts {
		o(s)
	}

	s.Server = &http.Server{
		TLSConfig: s.tlsConf,
	}

	http.HandleFunc(s.path, s.wsHandler)
}

func (s *Server) ClientCount() int {
	return len(s.clients)
}

func (s *Server) SendMessage(connectionId string, message *Message) {
	c, ok := s.clients[connectionId]
	if ok {
		c.SendMessage(message)
	}
}

func (s *Server) Broadcast(message *Message) {
	for _, c := range s.clients {
		c.SendMessage(message)
	}
}

func (s *Server) wsHandler(res http.ResponseWriter, req *http.Request) {
	conn, err := s.upgrader.Upgrade(res, req, nil)
	if err != nil {
		s.log.Fatal("upgrade exception:", err)
		return
	}

	client := NewClient(conn, s)
	client.server.register <- client

	client.Listen()
}

func (s *Server) listen() error {
	if s.lis == nil {
		lis, err := net.Listen(s.network, s.address)
		if err != nil {
			return err
		}
		s.lis = lis
	}

	addr := s.address
	if s.tlsConf == nil {
		if !strings.HasPrefix(addr, "ws://") {
			addr = "ws://" + addr
		}
	} else {
		if !strings.HasPrefix(addr, "wss://") {
			addr = "wss://" + addr
		}
	}

	s.endpoint, s.err = url.Parse(addr)

	return nil
}

func (s *Server) Endpoint() (*url.URL, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.endpoint, nil
}

func (s *Server) run() {
	for {
		select {
		case client := <-s.register:
			s.addClient(client)
		case client := <-s.unregister:
			s.removeClient(client)
		}
	}
}

func (s *Server) Start(ctx context.Context) error {
	if s.err != nil {
		return s.err
	}
	s.BaseContext = func(net.Listener) context.Context {
		return ctx
	}
	s.log.Infof("[websocket] server listening on: %s", s.lis.Addr().String())

	go s.run()

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
	s.log.Info("[websocket] server stopping")
	return s.Shutdown(ctx)
}

func (s *Server) addClient(c *Client) {
	//s.log.Info("add client: ", c.ConnectionID())
	s.clients[c.ConnectionID()] = c

	if s.connectHandler != nil {
		s.connectHandler(c.ConnectionID(), true)
	}
}

func (s *Server) removeClient(c *Client) {
	for k, v := range s.clients {
		if c == v {
			//s.log.Info("remove client: ", c.ConnectionID())
			if s.connectHandler != nil {
				s.connectHandler(c.ConnectionID(), false)
			}
			delete(s.clients, k)
			return
		}
	}
}
