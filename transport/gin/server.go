package gin

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	kratosTransport "github.com/go-kratos/kratos/v2/transport"
	kHttp "github.com/go-kratos/kratos/v2/transport/http"

	"github.com/tx7do/kratos-transport/transport"
)

var (
	_ kratosTransport.Server     = (*Server)(nil)
	_ kratosTransport.Endpointer = (*Server)(nil)
)

type Server struct {
	*gin.Engine
	server      *http.Server
	httpHandler http.Handler

	tlsConf *tls.Config
	timeout time.Duration

	network  string
	address  string
	endpoint *url.URL
	lis      net.Listener

	err error

	stateMu sync.RWMutex
	serving bool

	filters []kHttp.FilterFunc
	ms      []middleware.Middleware
	dec     kHttp.DecodeRequestFunc
	enc     kHttp.EncodeResponseFunc
	ene     kHttp.EncodeErrorFunc
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		network: "tcp",
		timeout: 1 * time.Second,
		dec:     kHttp.DefaultRequestDecoder,
		enc:     kHttp.DefaultResponseEncoder,
		ene:     kHttp.DefaultErrorEncoder,
	}

	srv.init(opts...)

	return srv
}

func (s *Server) init(opts ...ServerOption) {
	s.Engine = gin.New()

	for _, o := range opts {
		o(s)
	}

	s.installMiddlewares()

	s.httpHandler = s.buildHandlerChain()
	s.server = s.buildHTTPServer()
}

// installMiddlewares 把 kratos 风格的选项接线到 gin 引擎：
// WithTimeout → 请求级 deadline；WithMiddleware → kratos 中间件适配；
// WithFilter → 以 http 中间件形式包裹整个引擎（在 buildHTTPServer 时生效）。
func (s *Server) installMiddlewares() {
	if s.timeout > 0 {
		s.Engine.Use(func(c *gin.Context) {
			ctx, cancel := context.WithTimeout(c.Request.Context(), s.timeout)
			defer cancel()
			c.Request = c.Request.WithContext(ctx)
			c.Next()
		})
	}

	for _, mw := range s.ms {
		m := mw
		s.Engine.Use(func(c *gin.Context) {
			tr := &Transport{
				operation:    c.FullPath(),
				request:      c.Request,
				pathTemplate: c.FullPath(),
			}
			tr.reqHeader = headerCarrier(c.Request.Header)
			// 预分配响应头载体：中间件写 ReplyHeader 时避免 nil map 赋值 panic
			tr.replyHeader = headerCarrier(http.Header{})

			ctx := kratosTransport.NewServerContext(c.Request.Context(), tr)

			// 终端 handler：继续执行 gin 后续链路（路由 handler 等）
			invoked := false
			handler := middleware.Handler(func(ctx context.Context, req any) (any, error) {
				invoked = true
				c.Next()
				return nil, nil
			})

			_, err := m(handler)(ctx, c.Request)

			// 中间件未放行（拒绝型：返回错误或静默吞掉请求）：
			// gin 的 handler 循环在中间件返回后会继续推进，
			// 必须显式 Abort 才能真正拦下路由 handler
			if !invoked {
				c.Abort()
				if err != nil && !c.Writer.Written() {
					_ = c.Error(err)
					s.ene(c.Writer, c.Request, err)
				}
				return
			}

			if err != nil {
				_ = c.Error(err)
			}
		})
	}
}

// buildHTTPServer 创建 http.Server，并把 kratos 的 Filter 包裹在引擎之外。
func (s *Server) buildHTTPServer() *http.Server {
	s.httpHandler = s.buildHandlerChain()

	return &http.Server{
		Addr:      s.address,
		Handler:   s.httpHandler,
		TLSConfig: s.tlsConf,
	}
}

// buildHandlerChain 组装引擎 + kratos Filter 链
func (s *Server) buildHandlerChain() http.Handler {
	var handler http.Handler = s.Engine
	for i := len(s.filters) - 1; i >= 0; i-- {
		handler = s.filters[i](handler)
	}
	return handler
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

		s.endpoint = transport.NewRegistryEndpoint(KindGin, addr)
	}

	return nil
}

func (s *Server) Start(_ context.Context) error {
	s.stateMu.Lock()
	if s.serving {
		s.stateMu.Unlock()
		return nil
	}
	s.serving = true
	s.stateMu.Unlock()

	defer func() {
		s.stateMu.Lock()
		s.serving = false
		s.stateMu.Unlock()
	}()

	if err := s.listenAndEndpoint(); err != nil {
		return err
	}

	LogInfof("server listening on: %s", s.address)

	// Stop 之后的 http.Server 已永久关闭，重启时必须换新的实例
	if s.server == nil {
		s.server = s.buildHTTPServer()
	}

	var err error
	if s.tlsConf != nil {
		err = s.server.ServeTLS(s.lis, "", "")
	} else {
		err = s.server.Serve(s.lis)
	}
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}

	LogInfo("server stopping...")

	err := s.server.Shutdown(ctx)

	// Shutdown 后 http.Server 不可复用；同时释放 listener 与 endpoint，
	// 使下一次 Start 能重新监听（重启支持）
	s.server = nil
	if s.lis != nil {
		_ = s.lis.Close()
		s.lis = nil
	}
	s.endpoint = nil
	s.err = nil

	LogInfo("server stopped.")

	return err
}

func (s *Server) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	// 与 Start 的监听路径一致：经过 kratos Filter 链
	s.httpHandler.ServeHTTP(res, req)
}
