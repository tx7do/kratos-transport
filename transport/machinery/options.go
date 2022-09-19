package machinery

import (
	"crypto/tls"
	"github.com/RichardKnop/machinery/v2/config"
	"github.com/RichardKnop/machinery/v2/tasks"
	"github.com/go-kratos/kratos/v2/log"
	"time"
)

type ServerOption func(o *Server)

func WithYamlConfig(cnfPath string, keepReloading bool) ServerOption {
	return func(s *Server) {
		cnf, err := config.NewFromYaml(cnfPath, keepReloading)
		if err != nil {
			log.Errorf("[machinery] load yaml config [%s] failed: %s", cnfPath, err.Error())
		}
		s.cfg = cnf
	}
}

func WithEnvironmentConfig() ServerOption {
	return func(s *Server) {
		cnf, err := config.NewFromEnvironment()
		if err != nil {
			log.Errorf("[machinery] load environment config failed: %s", err.Error())
		}
		s.cfg = cnf
	}
}

func WithTLSConfig(c *tls.Config) ServerOption {
	return func(s *Server) {
		if s.cfg == nil {
			s.cfg = &config.Config{}
		}
		s.cfg.TLSConfig = c
	}
}

// WithRedisAddress broker & backend address
func WithRedisAddress(brokers, backends []string) ServerOption {
	return func(s *Server) {
		if s.cfg == nil {
			s.cfg = &config.Config{
				DefaultQueue:    "kratos_tasks",
				ResultsExpireIn: 3600,
			}
		}
		if s.cfg.Redis == nil {
			s.cfg.Redis = &config.RedisConfig{
				MaxIdle:                3,
				IdleTimeout:            240,
				ReadTimeout:            15,
				WriteTimeout:           15,
				ConnectTimeout:         15,
				NormalTasksPollPeriod:  1000,
				DelayedTasksPollPeriod: 500,
			}
		}
		s.redisOption.brokers = brokers
		s.redisOption.backends = backends
	}
}

type TaskOption func(o *tasks.Signature)

func WithDelayTime(delayTime time.Time) TaskOption {
	return func(o *tasks.Signature) {
		o.ETA = &delayTime
	}
}

func WithRetryCount(count int) TaskOption {
	return func(o *tasks.Signature) {
		o.RetryCount = count
	}
}

func WithRetryTimeout(timeout int) TaskOption {
	return func(o *tasks.Signature) {
		o.RetryTimeout = timeout
	}
}

func WithHeaders(headers tasks.Headers) TaskOption {
	return func(o *tasks.Signature) {
		o.Headers = headers
	}
}

func WithHeader(key string, value interface{}) TaskOption {
	return func(o *tasks.Signature) {
		if o.Headers == nil {
			o.Headers = tasks.Headers{}
		}
		o.Headers[key] = value
	}
}

func WithRoutingKey(key string) TaskOption {
	return func(o *tasks.Signature) {
		o.RoutingKey = key
	}
}

func WithPriority(priority uint8) TaskOption {
	return func(o *tasks.Signature) {
		o.Priority = priority
	}
}

func WithArgument(typeName string, value interface{}) TaskOption {
	return func(o *tasks.Signature) {
		o.Args = append(o.Args, tasks.Arg{Type: typeName, Value: value})
	}
}

type TasksOption func(o *[]*tasks.Signature)

func WithTask(typeName string, opts ...TaskOption) TasksOption {
	return func(s *[]*tasks.Signature) {
		signature := &tasks.Signature{Name: typeName}
		for _, o := range opts {
			o(signature)
		}
		*s = append(*s, signature)
	}
}
