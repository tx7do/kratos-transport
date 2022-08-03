package websocket

import (
	"crypto/tls"
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

func WithConnectHandle(h ConnectHandler) ServerOption {
	return func(s *Server) {
		s.connectHandler = h
	}
}

func WithReadHandle(path string, h Handler) ServerOption {
	return func(s *Server) {
		s.path = path
		s.readHandler = h
	}
}

func WithEchoHandle(path string, h EchoHandler) ServerOption {
	return func(s *Server) {
		s.path = path
		s.echoHandler = h
	}
}

func WithLogger(logger log.Logger) ServerOption {
	return func(s *Server) {
		s.log = log.NewHelper(logger)
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
