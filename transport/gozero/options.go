package gozero

import (
	"net"
	"strconv"
	"time"
)

type ServerOption func(*Server)

func WithTLSConfig(certFile, keyFile string) ServerOption {
	return func(o *Server) {
		o.cfg.CertFile = certFile
		o.cfg.KeyFile = keyFile
	}
}

func WithAddress(addr string) ServerOption {
	return func(s *Server) {
		host, strPort, _ := net.SplitHostPort(addr)

		if host != "" {
			s.cfg.Host = host
		}
		if strPort != "" {
			port, _ := strconv.Atoi(strPort)
			s.cfg.Port = port
		}
	}
}

func WithTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.cfg.Timeout = timeout.Milliseconds()
	}
}

func WithMaxConnections(cnt int) ServerOption {
	return func(s *Server) {
		s.cfg.MaxConns = cnt
	}
}
