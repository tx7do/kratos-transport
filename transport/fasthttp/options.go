package fasthttp

import (
	"crypto/tls"
	"time"

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
		s.address = addr
	}
}

// Timeout 预留选项：当前不参与请求处理。
func WithTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.timeout = timeout
	}
}

// Middleware 预留选项：当前不参与请求处理。
func WithMiddleware(m ...middleware.Middleware) ServerOption {
	return func(o *Server) {
		o.ms = m
	}
}

func WithFilter(filters ...FilterFunc) ServerOption {
	return func(o *Server) {
		o.filters = filters
	}
}

// RequestDecoder 预留选项：当前不参与请求处理。
func WithRequestDecoder(dec kHttp.DecodeRequestFunc) ServerOption {
	return func(o *Server) {
		o.dec = dec
	}
}

// ResponseEncoder 预留选项：当前不参与请求处理。
func WithResponseEncoder(en kHttp.EncodeResponseFunc) ServerOption {
	return func(o *Server) {
		o.enc = en
	}
}

// ErrorEncoder 预留选项：当前不参与请求处理。
func WithErrorEncoder(en kHttp.EncodeErrorFunc) ServerOption {
	return func(o *Server) {
		o.ene = en
	}
}
