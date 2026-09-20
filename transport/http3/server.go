package http3

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/go-kratos/kratos/v2/middleware"
	kratosTransport "github.com/go-kratos/kratos/v2/transport"
	kHttp "github.com/go-kratos/kratos/v2/transport/http"

	"github.com/gorilla/mux"
	"github.com/quic-go/quic-go/http3"

	"github.com/tx7do/kratos-transport/transport"
)

var (
	_ kratosTransport.Server     = (*Server)(nil)
	_ kratosTransport.Endpointer = (*Server)(nil)
)

type Server struct {
	*http3.Server

	tlsConf  *tls.Config
	timeout  time.Duration
	endpoint *url.URL

	err error

	filters []kHttp.FilterFunc
	ms      []middleware.Middleware
	dec     kHttp.DecodeRequestFunc
	enc     kHttp.EncodeResponseFunc
	ene     kHttp.EncodeErrorFunc

	router      *mux.Router
	strictSlash bool

	stopped     bool
	everStarted bool
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		timeout:     1 * time.Second,
		dec:         kHttp.DefaultRequestDecoder,
		enc:         kHttp.DefaultResponseEncoder,
		ene:         kHttp.DefaultErrorEncoder,
		strictSlash: true,
	}

	srv.init(opts...)

	return srv
}

func (s *Server) init(opts ...ServerOption) {
	s.Server = &http3.Server{
		Addr: ":443",
	}

	for _, o := range opts {
		o(s)
	}

	if s.tlsConf == nil {
		s.tlsConf = s.generateTLSConfig()
	}
	s.Server.TLSConfig = s.tlsConf

	s.router = mux.NewRouter().StrictSlash(s.strictSlash)
	s.router.NotFoundHandler = http.NotFoundHandler()
	s.router.MethodNotAllowedHandler = http.NotFoundHandler()

	// Apply the request filter middleware (timeout control, transport injection)
	handler := s.filter()(s.router)

	s.Server.Handler = kHttp.FilterChain(s.filters...)(handler)
}

func (s *Server) Endpoint() (*url.URL, error) {
	if err := s.listenAndEndpoint(); err != nil {
		return nil, err
	}
	return s.endpoint, nil
}

func (s *Server) listenAndEndpoint() error {
	if s.endpoint == nil {
		host, port, err := net.SplitHostPort(s.Addr)
		if err != nil {
			return err
		}

		if host == "" {
			ip, _ := transport.GetLocalIP()
			host = ip
		}

		addr := host + ":" + fmt.Sprint(port)
		s.endpoint = transport.NewRegistryEndpoint("https", addr)
	}

	return nil
}

func (s *Server) Start(ctx context.Context) error {
	if s.stopped {
		// quic-go 的 http3.Server 一旦 Close/Shutdown 便永久失效，
		// 静默重启只会得到“假启动”，这里显式报错
		return errors.New("http3 server cannot be restarted after Stop; create a new server instance")
	}

	if s.err = s.listenAndEndpoint(); s.err != nil {
		return s.err
	}

	s.everStarted = true

	LogInfof("server listening on: %s", s.Addr)

	if err := s.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			LogErrorf("start server failed: %s", err.Error())
			return err
		}
	}

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	if s.everStarted {
		s.stopped = true
	}

	LogInfo("server stopping...")

	// 优先优雅关闭（发 GOAWAY、等待在途请求），
	// ctx 取消或超时后降级为硬关闭
	err := s.Shutdown(ctx)
	if err != nil {
		LogWarnf("graceful shutdown failed, closing: %s", err.Error())
		err = s.Close()
	}
	s.err = nil

	LogInfo("server stopped.")

	return err
}

func (s *Server) Route(prefix string, filters ...kHttp.FilterFunc) *Router {
	return newRouter(prefix, s, filters...)
}

func (s *Server) Handle(path string, h http.Handler) {
	s.router.Handle(path, h)
}

func (s *Server) HandlePrefix(prefix string, h http.Handler) {
	s.router.PathPrefix(prefix).Handler(h)
}

func (s *Server) HandleFunc(path string, h http.HandlerFunc) {
	s.router.HandleFunc(path, h)
}

func (s *Server) HandleHeader(key, val string, h http.HandlerFunc) {
	s.router.Headers(key, val).Handler(h)
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
				// /path/123 -> /path/{id}
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

func (s *Server) generateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"h3"},
	}
}
