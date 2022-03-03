package websocket

import (
	"context"
	"crypto/tls"
	"errors"
	"github.com/gorilla/websocket"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/gorilla/mux"
)

type SendBuffer []byte
type SendBufferArray []SendBuffer
type Handler func(int, []byte) (SendBufferArray, error)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 解决跨域问题
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var (
	_ transport.Server     = (*Server)(nil)
	_ transport.Endpointer = (*Server)(nil)
)

// ServerOption is kafka server option.
type ServerOption func(o *Server)

// Network with server network.
func Network(network string) ServerOption {
	return func(s *Server) {
		s.network = network
	}
}

// Address with server address.
func Address(addr string) ServerOption {
	return func(s *Server) {
		s.address = addr
	}
}

// Timeout with server timeout.
func Timeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.timeout = timeout
	}
}

func Handle(path string, h Handler) ServerOption {
	return func(s *Server) {
		s.path = path
		s.handler = h
	}
}

// Logger with server logger.
func Logger(logger log.Logger) ServerOption {
	return func(s *Server) {
		s.log = log.NewHelper(logger)
	}
}

// TLSConfig with TLS config.
func TLSConfig(c *tls.Config) ServerOption {
	return func(o *Server) {
		o.tlsConf = c
	}
}

// StrictSlash is with mux's StrictSlash
// If true, when the path pattern is "/path/", accessing "/path" will
// redirect to the former and vice versa.
func StrictSlash(strictSlash bool) ServerOption {
	return func(o *Server) {
		o.strictSlash = strictSlash
	}
}

// Listener with server lis
func Listener(lis net.Listener) ServerOption {
	return func(s *Server) {
		s.lis = lis
	}
}

// Server is a kafka server wrapper.
type Server struct {
	*http.Server
	lis         net.Listener
	tlsConf     *tls.Config
	endpoint    *url.URL
	err         error
	network     string
	address     string
	timeout     time.Duration
	strictSlash bool
	router      *mux.Router
	log         *log.Helper
	handler     Handler
	path        string
}

// NewServer creates a kafka server by options.
func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		network:     "tcp",
		address:     ":0",
		timeout:     1 * time.Second,
		strictSlash: true,
		log:         log.NewHelper(log.GetLogger()),
	}

	for _, o := range opts {
		o(srv)
	}

	srv.router = mux.NewRouter().StrictSlash(srv.strictSlash)

	srv.HandleFunc(srv.path, srv.WsHandler)

	srv.Server = &http.Server{
		Handler:   srv.router,
		TLSConfig: srv.tlsConf,
	}

	srv.router.PathPrefix("/").Handler(srv.router)

	srv.err = srv.listenAndEndpoint()

	return srv
}

// Handle registers a new route with a matcher for the URL path.
func (s *Server) Handle(path string, h http.Handler) {
	s.router.Handle(path, h)
}

// HandleFunc registers a new route with a matcher for the URL path.
func (s *Server) HandleFunc(path string, h http.HandlerFunc) {
	s.router.HandleFunc(path, h)
}

func (s *Server) WsHandler(res http.ResponseWriter, req *http.Request) {
	c, err := upgrader.Upgrade(res, req, nil)
	if err != nil {
		s.log.Fatal("upgrade exception:", err)
		return
	}
	defer func(c *websocket.Conn) {
		err := c.Close()
		if err != nil {
			s.log.Fatal("close websocket connection exception:", err)
		}
	}(c)

	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			s.log.Fatal("read websocket message exception:", err)
			break
		}

		sendBuffers, err := s.handler(mt, message)
		if err != nil {
			s.log.Fatal("handle websocket message exception:", err)
			break
		}
		if sendBuffers != nil {
			for _, msg := range sendBuffers {
				err = c.WriteMessage(mt, msg)
				if err != nil {
					s.log.Fatal("send websocket message exception:", err)
					break
				}
			}
		}
	}
}

// Endpoint return a real address to registry endpoint.
// examples:
//   http://127.0.0.1:8000?isSecure=false
func (s *Server) Endpoint() (*url.URL, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.endpoint, nil
}

// Start the WS server.
func (s *Server) Start(ctx context.Context) error {
	if s.err != nil {
		return s.err
	}
	s.BaseContext = func(net.Listener) context.Context {
		return ctx
	}
	s.log.Infof("[WS] server listening on: %s", s.lis.Addr().String())
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

// Stop the WS server.
func (s *Server) Stop(ctx context.Context) error {
	s.log.Info("[WS] server stopping")
	return s.Shutdown(ctx)
}

func (s *Server) listenAndEndpoint() error {
	if s.lis == nil {
		lis, err := net.Listen(s.network, s.address)
		if err != nil {
			return err
		}
		s.lis = lis
	}

	//var err error
	//s.endpoint, err = url.Parse(s.address)
	//if err != nil {
	//	s.log.Errorf("error Endpoint: %X", err)
	//}

	return nil
}
