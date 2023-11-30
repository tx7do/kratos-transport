package kafka

import (
	"crypto/tls"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/broker/kafka"
)

type ServerOption func(o *Server)

// WithBrokerOptions MQ代理配置
func WithBrokerOptions(opts ...broker.Option) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, opts...)
	}
}

// WithAddress MQ代理地址
func WithAddress(addrs []string) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, broker.WithAddress(addrs...))
	}
}

// WithTLSConfig TLS配置
func WithTLSConfig(c *tls.Config) ServerOption {
	return func(s *Server) {
		if c != nil {
			s.brokerOpts = append(s.brokerOpts, broker.WithEnableSecure(true))
		}
		s.brokerOpts = append(s.brokerOpts, broker.WithTLSConfig(c))
	}
}

// WithCodec 编解码器
func WithCodec(c string) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, broker.WithCodec(c))
	}
}

// WithEnableKeepAlive enable keep alive
func WithEnableKeepAlive(enable bool) ServerOption {
	return func(s *Server) {
		s.enableKeepAlive = enable
	}
}

// WithPlainMechanism PLAIN认证信息
func WithPlainMechanism(username, password string) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, kafka.WithPlainMechanism(username, password))
	}
}

// WithScramMechanism SCRAM认证信息
func WithScramMechanism(algo string, username, password string) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, kafka.WithScramMechanism(kafka.ScramAlgorithm(algo), username, password))
	}
}

// WithGlobalTracerProvider 注入全局的链路追踪器的Provider
func WithGlobalTracerProvider() ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, broker.WithGlobalTracerProvider())
	}
}

// WithGlobalPropagator 注入全局的链路追踪器的Propagator
func WithGlobalPropagator() ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, broker.WithGlobalPropagator())
	}
}

// WithTracerProvider 注入链路追踪器的Provider
func WithTracerProvider(provider trace.TracerProvider, tracerName string) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, broker.WithTracerProvider(provider, tracerName))
	}
}

// WithPropagator 注入链路追踪器的Propagator
func WithPropagator(propagators propagation.TextMapPropagator) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, broker.WithPropagator(propagators))
	}
}
