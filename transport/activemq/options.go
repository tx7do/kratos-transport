package activemq

import (
	"crypto/tls"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/codec"
)

type ServerOption func(o *Server)

func WithAddress(addrs []string) ServerOption {
	return func(s *Server) {
		s.bOpts = append(s.bOpts, broker.Addrs(addrs...))
	}
}

func WithLogger(logger log.Logger) ServerOption {
	return func(s *Server) {
		s.log = log.NewHelper(logger)
	}
}

func WithTLSConfig(c *tls.Config) ServerOption {
	return func(s *Server) {
		if c != nil {
			s.bOpts = append(s.bOpts, broker.Secure(true))
		}
		s.bOpts = append(s.bOpts, broker.TLSConfig(c))
	}
}

func WithCodec(c codec.Marshaler) ServerOption {
	return func(s *Server) {
		s.bOpts = append(s.bOpts, broker.Codec(c))
	}
}
