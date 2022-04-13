package kafka

import (
	"context"
	"crypto/tls"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-transport/broker"
)

type ServerOption func(o *Server)

func Address(addr string) ServerOption {
	return func(s *Server) {
		s.bOpts = append(s.bOpts, broker.Addrs(addr))
	}
}

func Logger(logger log.Logger) ServerOption {
	return func(s *Server) {
		s.log = log.NewHelper(logger)
	}
}

func TLSConfig(c *tls.Config) ServerOption {
	return func(s *Server) {
		if c != nil {
			s.bOpts = append(s.bOpts, broker.Secure(true))
		}
		s.bOpts = append(s.bOpts, broker.TLSConfig(c))
	}
}

func Subscribe(ctx context.Context, topic, queue string, disableAutoAck bool, h broker.Handler) ServerOption {
	return func(s *Server) {
		_ = s.RegisterSubscriber(ctx, topic, queue, disableAutoAck, h)
	}
}
