package redis

import (
	"crypto/tls"
	"time"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/broker/redis"
)

type ServerOption func(o *Server)

// WithBrokerOptions MQ代理配置
func WithBrokerOptions(opts ...broker.Option) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, opts...)
	}
}

// WithAddress Redis服务器地址
func WithAddress(addr string) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, broker.WithAddress(addr))
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

// WithEnableKeepAlive enable keep alive
func WithEnableKeepAlive(enable bool) ServerOption {
	return func(s *Server) {
		s.enableKeepAlive = enable
	}
}

// WithCodec 编解码器
func WithCodec(c string) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, broker.WithCodec(c))
	}
}

// WithConnectTimeout 连接Redis超时时间
func WithConnectTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, redis.WithConnectTimeout(timeout))
	}
}

// WithReadTimeout 从Redis读取数据超时时间
func WithReadTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, redis.WithReadTimeout(timeout))
	}
}

// WithWriteTimeout 向Redis写入数据超时时间
func WithWriteTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, redis.WithWriteTimeout(timeout))
	}
}

// WithIdleTimeout 最大的空闲连接等待时间，超过此时间后，空闲连接将被关闭。如果设置成0，空闲连接将不会被关闭。应该设置一个比redis服务端超时时间更短的时间。
func WithIdleTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, redis.WithIdleTimeout(timeout))
	}
}

// WithMaxIdle 最大的空闲连接数，表示即使没有redis连接时依然可以保持N个空闲的连接，而不被清除，随时处于待命状态。
func WithMaxIdle(n int) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, redis.WithMaxIdle(n))
	}
}

// WithMaxActive 最大的连接数，表示同时最多有N个连接。0表示不限制。
func WithMaxActive(n int) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, redis.WithMaxActive(n))
	}
}
