package websocket

import (
	"crypto/tls"
	"github.com/go-kratos/kratos/v2/log"
	"net"
	"time"
)

type ServerOption func(o *Server)

func Network(network string) ServerOption {
	return func(s *Server) {
		s.network = network
	}
}

func Address(addr string) ServerOption {
	return func(s *Server) {
		s.address = addr
	}
}

func Timeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.timeout = timeout
	}
}

func RegisterHandle(h RegisterHandler) ServerOption {
	return func(s *Server) {
		s.registerHandler = h
	}
}

func ReadHandle(path string, h EchoHandler) ServerOption {
	return func(s *Server) {
		s.path = path
		s.readHandler = h
	}
}

func Logger(logger log.Logger) ServerOption {
	return func(s *Server) {
		s.log = log.NewHelper(logger)
	}
}

func TLSConfig(c *tls.Config) ServerOption {
	return func(o *Server) {
		o.tlsConf = c
	}
}

func Listener(lis net.Listener) ServerOption {
	return func(s *Server) {
		s.lis = lis
	}
}
