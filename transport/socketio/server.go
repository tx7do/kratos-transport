package socketio

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"
	"sync"

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

	router      *mux.Router
	checkOrigin func(*http.Request) bool

	// handler 记录：socket.io Server Close 后不可复用，
	// 重启重建实例时按记录重放全部 handler 注册
	handlersMu    sync.Mutex
	Registrations []handlerRegistration
	closed        bool
}

type handlerRegistration struct {
	kind      string // connect / disconnect / error / event
	namespace string
	event     string
	f         any
}

// createServer 用当前配置构造 socket.io 实例并重放已登记的 handler
func (s *Server) createServer() *socketIo.Server {
	server := socketIo.NewServer(&engineio.Options{
		Transports: []socketIoTransport.Transport{
			&polling.Transport{
				CheckOrigin: func(r *http.Request) bool { return s.checkOrigin(r) },
			},
			&websocket.Transport{
				CheckOrigin: func(r *http.Request) bool { return s.checkOrigin(r) },
			},
		},
	})

	s.handlersMu.Lock()
	for _, reg := range s.Registrations {
		switch reg.kind {
		case "connect":
			server.OnConnect(reg.namespace, reg.f.(func(socketIo.Conn) error))
		case "disconnect":
			server.OnDisconnect(reg.namespace, reg.f.(func(socketIo.Conn, string)))
		case "error":
			server.OnError(reg.namespace, reg.f.(func(socketIo.Conn, error)))
		case "event":
			server.OnEvent(reg.namespace, reg.event, reg.f)
		}
	}
	s.handlersMu.Unlock()

	return server
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

	// Close 后的 socket.io Server 不可复用（connChan 已关闭，新握手会
	// send-on-closed-channel panic）：重启时重建实例并重放 handler 注册
	// （路由上的委托 handler 会自动转发到新实例）
	if s.closed {
		s.Server = s.createServer()
		s.closed = false
	}

	go func() {
		if err := s.Server.Serve(); err != nil {
			LogErrorf("socketio serve error: %s", err.Error())
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

	// 关闭 HTTP listener，否则 http.Serve 不会返回、端口不会释放
	if s.lis != nil {
		_ = s.lis.Close()
		s.lis = nil
	}
	s.endpoint = nil
	s.closed = true
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
	s.recordHandler("connect", namespace, "", f)
	s.Server.OnConnect(namespace, f)
}

func (s *Server) RegisterDisconnectHandler(namespace string, f func(socketIo.Conn, string)) {
	s.recordHandler("disconnect", namespace, "", f)
	s.Server.OnDisconnect(namespace, f)
}

func (s *Server) RegisterErrorHandler(namespace string, f func(socketIo.Conn, error)) {
	s.recordHandler("error", namespace, "", f)
	s.Server.OnError(namespace, f)
}

func (s *Server) RegisterEventHandler(namespace, event string, f any) {
	s.recordHandler("event", namespace, event, f)
	s.Server.OnEvent(namespace, event, f)
}

func (s *Server) recordHandler(kind, namespace, event string, f any) {
	s.handlersMu.Lock()
	defer s.handlersMu.Unlock()
	s.Registrations = append(s.Registrations, handlerRegistration{kind: kind, namespace: namespace, event: event, f: f})
}

func (s *Server) init(opts ...ServerOption) {
	// 默认沿用旧行为（放行所有 Origin）；生产环境应通过 WithCheckOrigin 收紧
	if s.checkOrigin == nil {
		s.checkOrigin = func(r *http.Request) bool { return true }
	}

	// 必须先应用 opts 再创建 server：
	// Transport 的 CheckOrigin 在构造时捕获闭包，顺序反了会吞掉 WithCheckOrigin
	for _, o := range opts {
		o(s)
	}

	s.Server = s.createServer()
	if s.Server == nil {
		s.err = errors.New("create socket.io server failed")
		return
	}

	s.router.Use(mux.CORSMethodMiddleware(s.router))

	// 委托 handler：始终转发到当前 s.Server。
	// gorilla mux 的首条匹配路由生效且重复 Handle 不会覆盖旧路由，
	// 直接注册实例会导致重启后请求仍路由到已关闭的旧 server
	s.router.Handle(s.path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Server.ServeHTTP(w, r)
	}))
}
