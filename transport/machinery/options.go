package machinery

import (
	"crypto/tls"
	"github.com/RichardKnop/machinery/v2/config"
	"github.com/go-kratos/kratos/v2/log"
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
