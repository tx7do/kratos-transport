package webtransport

import (
	"crypto/tls"
	"github.com/go-kratos/kratos/v2/encoding"
	"time"
)

type ServerOption func(*Server)

func WithTLSConfig(c *tls.Config) ServerOption {
	return func(o *Server) {
		o.tlsConf = c
	}
}

func WithAddress(addr string) ServerOption {
	return func(s *Server) {
		s.Addr = addr
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

func WithCodec(c encoding.Codec) ServerOption {
	return func(s *Server) {
		s.codec = c
	}
}

////////////////////////////////////////////////////////////////////////////////

type ClientOption func(*Client)

func WithClientTLSConfig(c *tls.Config) ClientOption {
	return func(o *Client) {
		o.tlsConf = c
	}
}

func WithEndpoint(url string) ClientOption {
	return func(o *Client) {
		o.url = url
	}
}

func WithClientCodec(c encoding.Codec) ClientOption {
	return func(o *Client) {
		o.codec = c
	}
}
