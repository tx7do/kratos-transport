package signalr

import (
	"crypto/tls"
	"github.com/philippseith/signalr"
	"time"

	"github.com/go-kratos/kratos/v2/encoding"
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

func WithKeepAliveInterval(interval time.Duration) ServerOption {
	return func(s *Server) {
		s.keepAliveInterval = interval
	}
}

func WithChanReceiveTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.chanReceiveTimeout = timeout
	}
}

func WithStreamBufferCapacity(capacity uint) ServerOption {
	return func(s *Server) {
		s.streamBufferCapacity = capacity
	}
}

func WithDebug(enable bool) ServerOption {
	return func(s *Server) {
		s.debug = enable
	}
}

func WithHub(hub signalr.HubInterface) ServerOption {
	return func(s *Server) {
		s.hub = hub
	}
}

////////////////////////////////////////////////////////////////////////////////
