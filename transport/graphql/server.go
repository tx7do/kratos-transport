package graphql

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"

	"github.com/go-kratos/kratos/v2/middleware"
	kratosTransport "github.com/go-kratos/kratos/v2/transport"
	kHttp "github.com/go-kratos/kratos/v2/transport/http"

	"github.com/gorilla/mux"

	"github.com/tx7do/kratos-transport/transport"
)

var (
	_ kratosTransport.Server     = (*Server)(nil)
	_ kratosTransport.Endpointer = (*Server)(nil)
)

type Server struct {
	*http.Server

	lis      net.Listener
	tlsConf  *tls.Config
	endpoint *url.URL

	network     string
	address     string
	strictSlash bool
	timeout     time.Duration

	err error

	filters []kHttp.FilterFunc
	ms      []middleware.Middleware
	dec     kHttp.DecodeRequestFunc
	enc     kHttp.EncodeResponseFunc
	ene     kHttp.EncodeErrorFunc

	router *mux.Router
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		network:     "tcp",
		address:     ":0",
		timeout:     1 * time.Second,
		strictSlash: true,
		dec:         kHttp.DefaultRequestDecoder,
		enc:         kHttp.DefaultResponseEncoder,
		ene:         kHttp.DefaultErrorEncoder,
	}

	srv.init(opts...)

	return srv
}

func (s *Server) Name() string {
	return KindGraphQL
}

func (s *Server) Handle(path string, es graphql.ExecutableSchema) {
	s.router.Handle(path, handler.New(es))
}

func (s *Server) init(opts ...ServerOption) {
	for _, o := range opts {
		o(s)
	}

	s.router = mux.NewRouter().StrictSlash(s.strictSlash)
	// 未匹配请求显式 404：指向 DefaultServeMux 会把 /debug/pprof 等全局路由暴露出去
	s.router.NotFoundHandler = http.NotFoundHandler()
	s.router.MethodNotAllowedHandler = http.NotFoundHandler()

	// Apply the request filter middleware (timeout control, transport injection)
	handler := s.filter()(s.router)

	s.Server = &http.Server{
		TLSConfig: s.tlsConf,
		Handler:   kHttp.FilterChain(s.filters...)(handler),
	}
}

// rebuildServer 重建被 Shutdown 毒化的 http.Server（重启支持）
func (s *Server) rebuildServer() {
	handler := s.filter()(s.router)
	s.Server = &http.Server{
		TLSConfig: s.tlsConf,
		Handler:   kHttp.FilterChain(s.filters...)(handler),
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

	if s.endpoint == nil {
		// 如果传入的是完整的ip地址，则不需要调整。
		// 如果传入的只有端口号，则会调整为完整的地址，但，IP地址或许会不正确。
		addr, err := transport.AdjustAddress(s.address, s.lis)
		if err != nil {
			return err
		}

		scheme := "http"
		if s.tlsConf != nil {
			scheme = "https"
		}
		s.endpoint = transport.NewRegistryEndpoint(scheme, addr)
	}

	return nil
}

func (s *Server) Endpoint() (*url.URL, error) {
	if err := s.listenAndEndpoint(); err != nil {
		return nil, err
	}
	return s.endpoint, nil
}

func (s *Server) Start(ctx context.Context) error {
	if err := s.listenAndEndpoint(); err != nil {
		return err
	}

	s.BaseContext = func(net.Listener) context.Context {
		return ctx
	}
	LogInfof("server listening on: %s", s.lis.Addr().String())

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
	if s.Server == nil {
		return nil
	}

	LogInfo("server stopping...")

	err := s.Shutdown(ctx)

	// Shutdown 后 http.Server 不可复用：重建实例并释放 listener，支持 Stop→Start
	s.rebuildServer()
	if s.lis != nil {
		_ = s.lis.Close()
		s.lis = nil
	}
	s.endpoint = nil
	s.err = nil

	LogInfo("server stopped.")

	return err
}

func (s *Server) HandleFunc(path string, h http.HandlerFunc) {
	s.router.HandleFunc(path, h)
}

func (s *Server) HandlePrefix(prefix string, h http.Handler) {
	s.router.PathPrefix(prefix).Handler(h)
}

func (s *Server) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	s.Handler.ServeHTTP(res, req)
}

func (s *Server) filter() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			var (
				ctx    context.Context
				cancel context.CancelFunc
			)
			if s.timeout > 0 {
				ctx, cancel = context.WithTimeout(req.Context(), s.timeout)
			} else {
				ctx, cancel = context.WithCancel(req.Context())
			}
			defer cancel()

			pathTemplate := req.URL.Path
			if route := mux.CurrentRoute(req); route != nil {
				pathTemplate, _ = route.GetPathTemplate()
			}

			tr := &Transport{
				endpoint:     s.endpoint.String(),
				operation:    pathTemplate,
				reqHeader:    headerCarrier(req.Header),
				replyHeader:  headerCarrier(w.Header()),
				request:      req,
				pathTemplate: pathTemplate,
			}

			tr.request = req.WithContext(kratosTransport.NewServerContext(ctx, tr))
			next.ServeHTTP(w, tr.request)
		})
	}
}
