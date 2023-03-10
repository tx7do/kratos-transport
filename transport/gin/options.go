package gin

import (
	"crypto/tls"
	"github.com/gin-gonic/gin"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	kHttp "github.com/go-kratos/kratos/v2/transport/http"
)

type ServerOption func(*Server)

func WithTLSConfig(c *tls.Config) ServerOption {
	return func(o *Server) {
		o.tlsConf = c
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
	return func(o *Server) {
		o.ms = m
	}
}

func WithFilter(filters ...kHttp.FilterFunc) ServerOption {
	return func(o *Server) {
		o.filters = filters
	}
}

func WithRequestDecoder(dec kHttp.DecodeRequestFunc) ServerOption {
	return func(o *Server) {
		o.dec = dec
	}
}

func WithResponseEncoder(en kHttp.EncodeResponseFunc) ServerOption {
	return func(o *Server) {
		o.enc = en
	}
}

func WithErrorEncoder(en kHttp.EncodeErrorFunc) ServerOption {
	return func(o *Server) {
		o.ene = en
	}
}

func WithStrictSlash(strictSlash bool) ServerOption {
	return func(o *Server) {
		o.Engine.RedirectTrailingSlash = strictSlash
	}
}

// WithLogger inject info logger
func WithLogger(l log.Logger) ServerOption {
	return func(o *Server) {
		gin.DefaultWriter = &infoLogger{Logger: l}
		gin.DefaultErrorWriter = &errLogger{Logger: l}
	}
}
