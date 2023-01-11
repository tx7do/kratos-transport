package iris

import (
	"time"
)

type ServerOption func(*Server)

func WithTLSConfig(certFile, keyFile string) ServerOption {
	return func(o *Server) {
		o.certFile = certFile
		o.keyFile = keyFile
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
