package rocketmq

import (
	"crypto/tls"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/broker/rocketmq"
	"github.com/tx7do/kratos-transport/codec"
)

type ServerOption func(o *Server)

func WithTLSConfig(c *tls.Config) ServerOption {
	return func(s *Server) {
		if c != nil {
			s.bOpts = append(s.bOpts, broker.Secure(true))
		}
		s.bOpts = append(s.bOpts, broker.TLSConfig(c))
	}
}

func WithLogger(logger log.Logger) ServerOption {
	return func(s *Server) {
		s.log = log.NewHelper(logger)
	}
}

func WithAliyunHttpSupport() ServerOption {
	return func(s *Server) {
		s.bOpts = append(s.bOpts, rocketmq.WithAliyunHttpSupport())
	}
}

func WithEnableTrace() ServerOption {
	return func(s *Server) {
		s.bOpts = append(s.bOpts, rocketmq.WithEnableTrace())
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

func WithCredentials(accessKey, secretKey, securityToken string) ServerOption {
	return func(s *Server) {
		s.bOpts = append(s.bOpts, rocketmq.WithAccessKey(accessKey))
		s.bOpts = append(s.bOpts, rocketmq.WithSecretKey(secretKey))
		s.bOpts = append(s.bOpts, rocketmq.WithSecurityToken(securityToken))
	}
}

func WithNamespace(ns string) ServerOption {
	return func(s *Server) {
		s.bOpts = append(s.bOpts, rocketmq.WithNamespace(ns))
	}
}

func WithInstanceName(name string) ServerOption {
	return func(s *Server) {
		s.bOpts = append(s.bOpts, rocketmq.WithInstanceName(name))
	}
}

func WithGroupName(name string) ServerOption {
	return func(s *Server) {
		s.bOpts = append(s.bOpts, rocketmq.WithGroupName(name))
	}
}

func WithRetryCount(count int) ServerOption {
	return func(s *Server) {
		s.bOpts = append(s.bOpts, rocketmq.WithRetryCount(count))
	}
}

func WithCodec(c codec.Marshaler) ServerOption {
	return func(s *Server) {
		s.bOpts = append(s.bOpts, broker.Codec(c))
	}
}
