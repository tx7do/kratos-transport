package asynq

import (
	"crypto/tls"
	"time"

	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/hibiken/asynq"
)

const (
	defaultRedisAddress = "127.0.0.1:6379"
)

type ServerOption func(o *Server)

func WithRedisConnOpt(redisConnOpt asynq.RedisConnOpt) ServerOption {
	return func(s *Server) {
		s.redisConnOpt = redisConnOpt
	}
}

// WithEnableKeepAlive enable keep alive
func WithEnableKeepAlive(enable bool) ServerOption {
	return func(s *Server) {
		s.enableKeepAlive = enable
	}
}

// setRedisOption 是一个通用的辅助函数，用于设置 Redis 选项
func setRedisOption(opt asynq.RedisConnOpt, fn func(*asynq.RedisClientOpt, *asynq.RedisClusterClientOpt, *asynq.RedisFailoverClientOpt)) {
	if redisOpt, ok := opt.(*asynq.RedisClientOpt); ok {
		fn(redisOpt, nil, nil)
		return
	}
	if redisClusterOpt, ok := opt.(*asynq.RedisClusterClientOpt); ok {
		fn(nil, redisClusterOpt, nil)
		return
	}
	if redisFailoverOpt, ok := opt.(*asynq.RedisFailoverClientOpt); ok {
		fn(nil, nil, redisFailoverOpt)
	}
}

func WithRedisPoolSize(size int) ServerOption {
	return func(s *Server) {
		setRedisOption(s.redisConnOpt, func(redisOpt *asynq.RedisClientOpt, redisClusterOpt *asynq.RedisClusterClientOpt, redisFailoverOpt *asynq.RedisFailoverClientOpt) {
			if redisOpt != nil {
				redisOpt.PoolSize = size
			}
			if redisFailoverOpt != nil {
				redisFailoverOpt.PoolSize = size
			}
		})
	}
}

func WithDialTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		setRedisOption(s.redisConnOpt, func(redisOpt *asynq.RedisClientOpt, redisClusterOpt *asynq.RedisClusterClientOpt, redisFailoverOpt *asynq.RedisFailoverClientOpt) {
			if redisOpt != nil {
				redisOpt.DialTimeout = timeout
			}
			if redisClusterOpt != nil {
				redisClusterOpt.DialTimeout = timeout
			}
			if redisFailoverOpt != nil {
				redisFailoverOpt.DialTimeout = timeout
			}
		})
	}
}

func WithReadTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		setRedisOption(s.redisConnOpt, func(redisOpt *asynq.RedisClientOpt, redisClusterOpt *asynq.RedisClusterClientOpt, redisFailoverOpt *asynq.RedisFailoverClientOpt) {
			if redisOpt != nil {
				redisOpt.ReadTimeout = timeout
			}
			if redisClusterOpt != nil {
				redisClusterOpt.ReadTimeout = timeout
			}
			if redisFailoverOpt != nil {
				redisFailoverOpt.ReadTimeout = timeout
			}
		})
	}
}

func WithWriteTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		setRedisOption(s.redisConnOpt, func(redisOpt *asynq.RedisClientOpt, redisClusterOpt *asynq.RedisClusterClientOpt, redisFailoverOpt *asynq.RedisFailoverClientOpt) {
			if redisOpt != nil {
				redisOpt.WriteTimeout = timeout
			}
			if redisClusterOpt != nil {
				redisClusterOpt.WriteTimeout = timeout
			}
			if redisFailoverOpt != nil {
				redisFailoverOpt.WriteTimeout = timeout
			}
		})
	}
}

func WithTLSConfig(c *tls.Config) ServerOption {
	return func(s *Server) {
		setRedisOption(s.redisConnOpt, func(redisOpt *asynq.RedisClientOpt, redisClusterOpt *asynq.RedisClusterClientOpt, redisFailoverOpt *asynq.RedisFailoverClientOpt) {
			if redisOpt != nil {
				redisOpt.TLSConfig = c
			}
			if redisClusterOpt != nil {
				redisClusterOpt.TLSConfig = c
			}
			if redisFailoverOpt != nil {
				redisFailoverOpt.TLSConfig = c
			}
		})
	}
}

func WithConfig(cfg asynq.Config) ServerOption {
	return func(s *Server) {
		s.asynqConfig = cfg
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

func WithLogger(log *log.Helper) ServerOption {
	return func(s *Server) {
		s.schedulerOpts.Logger = log
	}
}

func WithLogLevel(log *log.Level) ServerOption {
	return func(s *Server) {
		_ = s.schedulerOpts.LogLevel.Set(log.String())
	}
}

func WithPreEnqueueFunc(fn func(task *asynq.Task, opts []asynq.Option)) ServerOption {
	return func(s *Server) {
		s.schedulerOpts.PreEnqueueFunc = fn
	}
}

func WithPostEnqueueFunc(fn func(info *asynq.TaskInfo, err error)) ServerOption {
	return func(s *Server) {
		s.schedulerOpts.PostEnqueueFunc = fn
	}
}

func WithCodec(c string) ServerOption {
	return func(s *Server) {
		s.codec = encoding.GetCodec(c)
	}
}

func WithIsFailure(c asynq.Config) ServerOption {
	return func(s *Server) {
		s.asynqConfig.IsFailure = c.IsFailure
	}
}

func WithShutdownTimeout(t time.Duration) ServerOption {
	return func(s *Server) {
		s.asynqConfig.ShutdownTimeout = t
	}
}

func WithGracefullyShutdown(enable bool) ServerOption {
	return func(s *Server) {
		s.gracefullyShutdown = enable
	}
}
