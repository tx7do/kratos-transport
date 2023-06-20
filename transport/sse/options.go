package sse

import (
	"crypto/tls"
	"net"
	"time"

	"github.com/go-kratos/kratos/v2/encoding"
)

const DefaultBufferSize = 1024

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

func WithBufferSize(size int) ServerOption {
	return func(s *Server) {
		s.bufferSize = size
	}
}

func WithCodec(c string) ServerOption {
	return func(s *Server) {
		s.codec = encoding.GetCodec(c)
	}
}

////////////////////////////////////////////////////////////////////////////////

type ClientOption func(o *Client)

func WithEndpoint(uri string) ClientOption {
	return func(c *Client) {
		c.url = uri
	}
}
