package asynq

import (
	"crypto/tls"
	"github.com/go-kratos/kratos/v2/encoding"
	"time"

	"github.com/hibiken/asynq"
)

const (
	defaultRedisAddress = "127.0.0.1:6379"
)

type ServerOption func(o *Server)

func WithAddress(addr string) ServerOption {
	return func(s *Server) {
		s.redisOpt.Addr = addr
	}
}

// WithEnableKeepAlive enable keep alive
func WithEnableKeepAlive(enable bool) ServerOption {
	return func(s *Server) {
		s.enableKeepAlive = enable
	}
}

func WithRedisAuth(userName, password string) ServerOption {
	return func(s *Server) {
		s.redisOpt.Username = userName
		s.redisOpt.Password = password
	}
}

func WithRedisPassword(password string) ServerOption {
	return func(s *Server) {
		s.redisOpt.Password = password
	}
}

func WithRedisDatabase(db int) ServerOption {
	return func(s *Server) {
		s.redisOpt.DB = db
	}
}

func WithRedisPoolSize(size int) ServerOption {
	return func(s *Server) {
		s.redisOpt.PoolSize = size
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

func WithRetryDelayFunc(fn asynq.RetryDelayFunc) ServerOption {
	return func(s *Server) {
		s.asynqConfig.RetryDelayFunc = fn
	}
}

func WithStrictPriority(val bool) ServerOption {
	return func(s *Server) {
		s.asynqConfig.StrictPriority = val
	}
}

func WithErrorHandler(fn asynq.ErrorHandler) ServerOption {
	return func(s *Server) {
		s.asynqConfig.ErrorHandler = fn
	}
}

func WithHealthCheckFunc(fn func(error)) ServerOption {
	return func(s *Server) {
		s.asynqConfig.HealthCheckFunc = fn
	}
}

func WithHealthCheckInterval(tm time.Duration) ServerOption {
	return func(s *Server) {
		s.asynqConfig.HealthCheckInterval = tm
	}
}

func WithDelayedTaskCheckInterval(tm time.Duration) ServerOption {
	return func(s *Server) {
		s.asynqConfig.DelayedTaskCheckInterval = tm
	}
}

func WithGroupGracePeriod(tm time.Duration) ServerOption {
	return func(s *Server) {
		s.asynqConfig.GroupGracePeriod = tm
	}
}

func WithGroupMaxDelay(tm time.Duration) ServerOption {
	return func(s *Server) {
		s.asynqConfig.GroupMaxDelay = tm
	}
}

func WithGroupMaxSize(sz int) ServerOption {
	return func(s *Server) {
		s.asynqConfig.GroupMaxSize = sz
	}
}

func WithMiddleware(m ...asynq.MiddlewareFunc) ServerOption {
	return func(o *Server) {
		o.mux.Use(m...)
	}
}

func WithLocation(name string) ServerOption {
	return func(s *Server) {
		loc, _ := time.LoadLocation(name)
		s.schedulerOpts.Location = loc
	}
}

func WithCodec(c string) ServerOption {
	return func(s *Server) {
		s.codec = encoding.GetCodec(c)
	}
}
