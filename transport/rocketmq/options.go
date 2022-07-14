package rocketmq

import (
	"context"
	"crypto/tls"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/broker/rocketmq"
)

type ServerOption func(o *Server)

func TLSConfig(c *tls.Config) ServerOption {
	return func(s *Server) {
		if c != nil {
			s.bOpts = append(s.bOpts, broker.Secure(true))
		}
		s.bOpts = append(s.bOpts, broker.TLSConfig(c))
	}
}

func Logger(logger log.Logger) ServerOption {
	return func(s *Server) {
		s.log = log.NewHelper(logger)
	}
}

func WithNameServer(addrs []string) ServerOption {
	return func(s *Server) {
		s.bOpts = append(s.bOpts, rocketmq.WithNameServer(addrs))
	}
}

func WithNameServerDomain(uri string) ServerOption {
	return func(s *Server) {
		s.bOpts = append(s.bOpts, rocketmq.WithNameServerDomain(uri))
	}
}

func WithCredentials(accessKey, secretKey string) ServerOption {
	return func(s *Server) {
		s.bOpts = append(s.bOpts, rocketmq.WithAccessKey(accessKey))
		s.bOpts = append(s.bOpts, rocketmq.WithSecretKey(secretKey))
	}
}

func WithNamespace(ns string) ServerOption {
	return func(s *Server) {
		s.bOpts = append(s.bOpts, rocketmq.WithNamespace(ns))
	}
}

func WithRetryCount(count int) ServerOption {
	return func(s *Server) {
		s.bOpts = append(s.bOpts, rocketmq.WithRetryCount(count))
	}
}

func Subscribe(ctx context.Context, topic, groupName string, h broker.Handler, opts ...broker.SubscribeOption) ServerOption {
	return func(s *Server) {
		_ = s.RegisterSubscriber(ctx, topic, groupName, h, opts...)
	}
}
