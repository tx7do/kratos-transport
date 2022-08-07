package rabbitmq

import (
	"crypto/tls"

	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/broker/rabbitmq"
)

type ServerOption func(o *Server)

func WithAddress(addrs []string) ServerOption {
	return func(s *Server) {
		s.bOpts = append(s.bOpts, broker.WithAddress(addrs...))
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
			s.bOpts = append(s.bOpts, broker.WithEnableSecure(true))
		}
		s.bOpts = append(s.bOpts, broker.WithTLSConfig(c))
	}
}

func WithExchange(name string, durable bool) ServerOption {
	return func(s *Server) {
		s.bOpts = append(s.bOpts, rabbitmq.ExchangeName(name))
		if durable {
			s.bOpts = append(s.bOpts, rabbitmq.DurableExchange())
		}
	}
}

func WithCodec(c encoding.Codec) ServerOption {
	return func(s *Server) {
		s.bOpts = append(s.bOpts, broker.WithCodec(c))
	}
}
