package asynq

import (
	"crypto/tls"
	"time"

	"github.com/hibiken/asynq"
)

type ServerOption func(o *Server)

func WithAddress(addr string) ServerOption {
	return func(s *Server) {
		s.redisOpt.Addr = addr
	}
}

func WithRedisAuth(userName, password string) ServerOption {
	return func(s *Server) {
		s.redisOpt.Username = userName
		s.redisOpt.Password = password
	}
}

func WithRedisPoolSize(size int) ServerOption {
	return func(s *Server) {
		s.redisOpt.PoolSize = size
	}
}

func WithRedisDatabase(db int) ServerOption {
	return func(s *Server) {
		s.redisOpt.DB = db
	}
}

func WithDialTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.redisOpt.DialTimeout = timeout
	}
}

func WithReadTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.redisOpt.ReadTimeout = timeout
	}
}

func WithWriteTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.redisOpt.WriteTimeout = timeout
	}
}

func WithTLSConfig(c *tls.Config) ServerOption {
	return func(s *Server) {
		s.redisOpt.TLSConfig = c
	}
}

func WithConcurrency(concurrency int) ServerOption {
	return func(s *Server) {
		s.asynqConfig.Concurrency = concurrency
	}
}

func WithQueues(queues map[string]int) ServerOption {
	return func(s *Server) {
		s.asynqConfig.Queues = queues
	}
}

func WithMiddleware(m ...asynq.MiddlewareFunc) ServerOption {
	return func(o *Server) {
		o.mux.Use(m...)
	}
}
