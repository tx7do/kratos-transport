package mqtt

import (
	"crypto/tls"
	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/broker/mqtt"
)

type ServerOption func(o *Server)

func WithAddress(addrs []string) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, broker.WithAddress(addrs...))
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

func WithCleanSession(enable bool) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, mqtt.WithCleanSession(enable))
	}
}

func WithAuth(username string, password string) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, mqtt.WithAuth(username, password))
	}
}

func WithClientId(clientId string) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, mqtt.WithClientId(clientId))
	}
}

func WithCodec(c encoding.Codec) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, broker.WithCodec(c))
	}
}
