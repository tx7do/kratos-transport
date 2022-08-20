package redis

import (
	"crypto/tls"
	"time"

	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/broker/redis"
)

type ServerOption func(o *Server)

func WithAddress(addr string) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, broker.WithAddress(addr))
	}
}

func WithReadTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, redis.ReadTimeout(timeout))
	}
}

func WithIdleTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, redis.IdleTimeout(timeout))
	}
}

func WithTLSConfig(c *tls.Config) ServerOption {
	return func(s *Server) {
		if c != nil {
			s.brokerOpts = append(s.brokerOpts, broker.WithEnableSecure(true))
		}
		s.brokerOpts = append(s.brokerOpts, broker.WithTLSConfig(c))
	}
}

func WithCodec(c encoding.Codec) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, broker.WithCodec(c))
	}
}
