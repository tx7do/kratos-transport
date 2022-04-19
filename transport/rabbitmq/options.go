package rabbitmq

import (
	"context"
	"crypto/tls"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/broker/rabbitmq"
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

func Subscribe(ctx context.Context, topic string, h broker.Handler, opts ...broker.SubscribeOption) ServerOption {
	return func(s *Server) {
		if ctx != nil {
			s.baseCtx = ctx
		}
		if s.baseCtx == nil {
			s.baseCtx = context.Background()
			ctx = s.baseCtx
		}

		opts = append(opts, broker.SubscribeContext(ctx))

		_ = s.RegisterSubscriber(topic, h, opts...)
	}
}

func SubscribeDurableQueue(topic, queue string, h broker.Handler) ServerOption {
	return func(s *Server) {
		if s.baseCtx == nil {
			s.baseCtx = context.Background()
		}

		_ = s.RegisterSubscriber(topic, h,
			broker.SubscribeContext(s.baseCtx),
			broker.Queue(queue),
			rabbitmq.DurableQueue(),
		)
	}
}
