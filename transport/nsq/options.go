package nsq

import (
	"crypto/tls"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/broker/nsq"
)

type ServerOption func(o *Server)

// WithBrokerOptions MQ代理配置
func WithBrokerOptions(opts ...broker.Option) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, opts...)
	}
}

func WithAddress(addrs []string) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, broker.WithAddress(addrs...))
	}
}

func WithLookupdAddress(addrs []string) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, nsq.WithLookupdAddress(addrs))
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

func WithCodec(c string) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, broker.WithCodec(c))
	}
}

func WithConsumerOptions(opts []string) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, nsq.WithConsumerOptions(opts))
	}
}
