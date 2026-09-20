package socketio

import (
	"crypto/tls"
	"net/http"

	"github.com/go-kratos/kratos/v2/encoding"
	socketIo "github.com/googollee/go-socket.io"
)

type ServerOption func(o *Server)

func WithNetwork(network string) ServerOption {
	return func(s *Server) {
		s.network = network
	}
}

func WithAddress(addr string) ServerOption {
	return func(s *Server) {
		s.address = addr
	}
}

func WithTLSConfig(c *tls.Config) ServerOption {
	return func(o *Server) {
		o.tlsConf = c
	}
}

func WithCodec(c string) ServerOption {
	return func(s *Server) {
		s.codec = encoding.GetCodec(c)
	}
}

func WithPath(path string) ServerOption {
	return func(s *Server) {
		s.path = path
	}
}

func WithConnectHandler(namespace string, f func(socketIo.Conn) error) ServerOption {
	return func(s *Server) {
		s.Server.OnConnect(namespace, f)
	}
}

func WithDisconnectHandler(namespace string, f func(socketIo.Conn, string)) ServerOption {
	return func(s *Server) {
		s.Server.OnDisconnect(namespace, f)
	}
}

func WithErrorHandler(namespace string, f func(socketIo.Conn, error)) ServerOption {
	return func(s *Server) {
		s.Server.OnError(namespace, f)
	}
}

func WithEventHandler(namespace, event string, f any) ServerOption {
	return func(s *Server) {
		s.Server.OnEvent(namespace, event, f)
	}
}

////////////////////////////////////////////////////////////////////////////////

// WithCheckOrigin 自定义 Origin 校验（默认放行所有 Origin）。
func WithCheckOrigin(fn func(r *http.Request) bool) ServerOption {
	return func(s *Server) {
		s.checkOrigin = fn
	}
}
