package hertz

import (
	"crypto/tls"
	"time"

	hertz "github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/config"

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

// WithStrictSlash 对齐 gin 的 StrictSlash 语义：true 时路由不匹配时按尾斜杠
// 重定向（301），false 时关闭。经 hertz.WithRedirectTrailingSlash 透传；
// 注意 hertz 默认开启重定向，显式传 false 可关闭。
func WithStrictSlash(strictSlash bool) ServerOption {
	return func(o *Server) {
		o.options = append(o.options, hertz.WithRedirectTrailingSlash(strictSlash))
	}
}

// WithServerOptions 透传 hertz 引擎级配置项，
// 例如 hertz.WithMaxRequestBodySize(16 << 20) 放大请求体大小限制（默认 4MB）。
// 透传项在 WithHostPorts/WithTLS 之后应用，可覆盖默认行为；可多次调用，依次追加。
func WithServerOptions(opts ...config.Option) ServerOption {
	return func(o *Server) {
		o.options = append(o.options, opts...)
	}
}
