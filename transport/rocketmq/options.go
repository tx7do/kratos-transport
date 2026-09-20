package rocketmq

import (
	"crypto/tls"

	rocketmqOption "github.com/tx7do/kratos-transport/broker/rocketmq/option"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/tx7do/kratos-transport/broker"
)

type ServerOption func(o *Server)

// WithBrokerOptions MQ代理配置
func WithBrokerOptions(opts ...broker.Option) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, opts...)
	}
}

// WithTLSConfig 注意：对应 broker 驱动当前不消费 TLS 配置，透传不生效。
func WithTLSConfig(c *tls.Config) ServerOption {
	return func(s *Server) {
		if c != nil {
			s.brokerOpts = append(s.brokerOpts, broker.WithEnableSecure(true))
		}
		s.brokerOpts = append(s.brokerOpts, broker.WithTLSConfig(c))
	}
}

func WithEnableTrace() ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, rocketmqOption.WithEnableTrace())
	}
}

func WithNameServer(addrs []string) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, rocketmqOption.WithNameServer(addrs))
	}
}

func WithNameServerDomain(uri string) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, rocketmqOption.WithNameServerDomain(uri))
	}
}

func WithCredentials(accessKey, secretKey, securityToken string) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, rocketmqOption.WithCredentials(accessKey, secretKey, securityToken))
	}
}

func WithNamespace(ns string) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, rocketmqOption.WithNamespace(ns))
	}
}

func WithInstanceName(name string) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, rocketmqOption.WithInstanceName(name))
	}
}

func WithGroupName(name string) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, rocketmqOption.WithGroupName(name))
	}
}

func WithRetryCount(count int) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, rocketmqOption.WithRetryCount(count))
	}
}

func WithCodec(c string) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, broker.WithCodec(c))
	}
}

func WithGlobalTracerProvider() ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, broker.WithGlobalTracerProvider())
	}
}

func WithGlobalPropagator() ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, broker.WithGlobalPropagator())
	}
}

func WithTracerProvider(provider trace.TracerProvider, _ string) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, broker.WithTracerProvider(provider))
	}
}

func WithPropagator(propagators propagation.TextMapPropagator) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, broker.WithPropagator(propagators))
	}
}
