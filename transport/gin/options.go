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
		s.addr = addr
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

func WithRequestDecoder(dec kHttp.DecodeRequestFunc) ServerOption {
	return func(s *Server) {
		s.dec = dec
	}
}

func WithResponseEncoder(en kHttp.EncodeResponseFunc) ServerOption {
	return func(s *Server) {
		s.enc = en
	}
}

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
