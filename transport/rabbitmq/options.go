package rabbitmq

import (
	"context"
	"crypto/tls"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/broker/rabbitmq"
)

type ServerOption func(o *Server)

func Address(addrs []string) ServerOption {
	return func(s *Server) {
		s.bOpts = append(s.bOpts, broker.Addrs(addrs...))
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

func Exchange(name string, durable bool) ServerOption {
	return func(s *Server) {
		s.bOpts = append(s.bOpts, rabbitmq.ExchangeName(name))
		if durable {
			s.bOpts = append(s.bOpts, rabbitmq.DurableExchange())
		}
	}
}

func Subscribe(ctx context.Context, routingKey string, h broker.Handler, opts ...broker.SubscribeOption) ServerOption {
	return func(s *Server) {
		_ = s.RegisterSubscriber(ctx, routingKey, h, opts...)
	}
}

func SubscribeDurableQueue(ctx context.Context, routingKey, queue string, h broker.Handler, opts ...broker.SubscribeOption) ServerOption {
	return func(s *Server) {
		opts = append(opts, broker.Queue(queue))
		opts = append(opts, rabbitmq.DurableQueue())
		_ = s.RegisterSubscriber(ctx, routingKey, h, opts...)
	}
}
