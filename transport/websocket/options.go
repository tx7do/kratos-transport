package websocket

import (
	"crypto/tls"
	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/go-kratos/kratos/v2/log"
	"net"
	"time"
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

func WithTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.timeout = timeout
	}
}

func WithPath(path string) ServerOption {
	return func(s *Server) {
		s.path = path
	}
}

func WithConnectHandle(h ConnectHandler) ServerOption {
	return func(s *Server) {
		s.connectHandler = h
	}
}

func WithLogger(logger log.Logger) ServerOption {
	return func(s *Server) {
		s.log = log.NewHelper(logger, log.WithMessageKey("websocket"))
	}
}

func WithTLSConfig(c *tls.Config) ServerOption {
	return func(o *Server) {
		o.tlsConf = c
	}
}

func WithListener(lis net.Listener) ServerOption {
	return func(s *Server) {
		s.lis = lis
	}
}

func WithCodec(c encoding.Codec) ServerOption {
	return func(s *Server) {
		s.Codec = c
	}
}
