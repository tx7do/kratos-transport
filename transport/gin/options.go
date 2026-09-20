package gin

import (
	"crypto/tls"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	kHttp "github.com/go-kratos/kratos/v2/transport/http"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type ServerOption func(*Server)

func WithTLSConfig(c *tls.Config) ServerOption {
	return func(s *Server) {
		s.tlsConf = c
	}
}

func WithAddress(addr string) ServerOption {
	return func(s *Server) {
		s.address = addr
	}
}

func WithTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.timeout = timeout
	}
}

func WithMiddleware(m ...middleware.Middleware) ServerOption {
	return func(s *Server) {
		s.ms = m
	}
}

func WithFilter(filters ...kHttp.FilterFunc) ServerOption {
	return func(s *Server) {
		s.filters = filters
	}
}

// WithRequestDecoder 预留选项：请求解码需要与路由的目标类型绑定，
// gin 路由由用户自行处理参数解析，该选项当前不参与请求处理。
func WithRequestDecoder(dec kHttp.DecodeRequestFunc) ServerOption {
	return func(s *Server) {
		s.dec = dec
	}
}

// WithResponseEncoder 预留选项：响应编码由 gin 路由自行完成，该选项当前不参与响应处理。
func WithResponseEncoder(en kHttp.EncodeResponseFunc) ServerOption {
	return func(s *Server) {
		s.enc = en
	}
}

// WithErrorEncoder 错误编码器：kratos 中间件链返回错误且尚未写响应时，用它输出错误响应。
func WithErrorEncoder(en kHttp.EncodeErrorFunc) ServerOption {
	return func(s *Server) {
		s.ene = en
	}
}

func WithStrictSlash(strictSlash bool) ServerOption {
	return func(s *Server) {
		s.Engine.RedirectTrailingSlash = strictSlash
	}
}

// WithLogger inject info logger
func WithLogger(l log.Logger) ServerOption {
	return func(s *Server) {
		gin.DefaultWriter = &infoLogger{Logger: l}
		gin.DefaultErrorWriter = &errLogger{Logger: l}
		s.Engine.Use(GinLogger(l), GinRecovery(l, true))
	}
}

// WithGlobalTracer 注入全局的链路追踪器
func WithGlobalTracer() ServerOption {
	return func(s *Server) {
		s.Engine.Use(otelgin.Middleware("gin",
			otelgin.WithTracerProvider(otel.GetTracerProvider()),
			otelgin.WithPropagators(otel.GetTextMapPropagator()),
		))
	}
}

// WithCustomTracer 注入链路追踪器
func WithCustomTracer(provider trace.TracerProvider, propagator propagation.TextMapPropagator) ServerOption {
	return func(s *Server) {
		s.Engine.Use(otelgin.Middleware("gin",
			otelgin.WithTracerProvider(provider),
			otelgin.WithPropagators(propagator),
		))
	}
}
