package asynq

import (
	"crypto/tls"
	"time"

	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/hibiken/asynq"
)

const (
	RedisTypeSingle   = "single"
	RedisTypeCluster  = "cluster"
	RedisTypeSentinel = "sentinel"
)

const (
	defaultRedisAddress = "127.0.0.1:6379"
	defaultRedisDB      = 0
	defaultConcurrency  = 20
)

func newRedisClientOpt() asynq.RedisConnOpt {
	return &asynq.RedisClientOpt{
		Addr: defaultRedisAddress,
		DB:   defaultRedisDB,
	}
}

func newRedisClusterClientOpt() asynq.RedisConnOpt {
	return &asynq.RedisClusterClientOpt{
		Addrs: []string{defaultRedisAddress},
	}
}

func newRedisFailoverClientOpt() asynq.RedisConnOpt {
	return &asynq.RedisFailoverClientOpt{
		MasterName:    "mymaster",
		SentinelAddrs: []string{defaultRedisAddress},
		DB:            defaultRedisDB,
	}
}

type ServerOption func(o *Server)

func WithRedisType(redisType string) ServerOption {
	return func(s *Server) {
		switch redisType {
		case RedisTypeSingle:
			s.redisConnOpt = newRedisClientOpt()
		case RedisTypeCluster:
			s.redisConnOpt = newRedisClusterClientOpt()
		case RedisTypeSentinel:
			s.redisConnOpt = newRedisFailoverClientOpt()
		default:
			panic("unknown redis type " + redisType)
		}
	}
}

func WithRedisConnOpt(redisConnOpt asynq.RedisConnOpt) ServerOption {
	return func(s *Server) {
		s.redisConnOpt = redisConnOpt
	}
}

// WithRedisURI 设置 Redis 连接 URI
// 支持三种类型的 Redis 连接：单节点、集群和哨兵模式
// URI 格式如下：
// 单节点模式: redis://[:password@]host:port[/db]。比如："redis://127.0.0.1:6379"
// 集群模式: redis+cluster://[:password@]host1:port1,host2:port2,...[/db]。比如："redis+cluster://127.0.0.1:6379,127.0.0.2:6379"
// 哨兵模式: redis+sentinel://[:password@]host1:port1,host2:port2,.../mastername[/db]。比如："redis+sentinel://mymaster@127.0.0.1:26379,127.0.0.2:26379"
func WithRedisURI(uri string) ServerOption {
	return func(s *Server) {
		redisConnOpt, err := asynq.ParseRedisURI(uri)
		if err != nil {
			panic(err)
		}
		s.redisConnOpt = redisConnOpt
	}
}

// WithRedisAddress 设置 Redis 地址，格式为 "host:port"
func WithRedisAddress(address string) ServerOption {
	return func(s *Server) {
		s.addresses = []string{address}
	}
}

// WithRedisAddresses 设置 Redis 地址列表
func WithRedisAddresses(addresses []string) ServerOption {
	return func(s *Server) {
		s.addresses = addresses
	}
}

func WithRedisUsername(username string) ServerOption {
	return func(s *Server) {
		s.username = &username
	}
}

func WithRedisPassword(password string) ServerOption {
	return func(s *Server) {
		s.password = &password
	}
}

func WithRedisAuth(username, password string) ServerOption {
	return func(s *Server) {
		s.username = &username
		s.password = &password
	}
}

func WithRedisDB(db int) ServerOption {
	return func(s *Server) {
		s.db = &db
	}
}

func WithRedisPoolSize(size int) ServerOption {
	return func(s *Server) {
		s.poolSize = &size
	}
}

func WithDialTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.dialTimeout = &timeout
	}
}

func WithReadTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.readTimeout = &timeout
	}
}

func WithWriteTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.writeTimeout = &timeout
	}
}

func WithTLSConfig(tlsConfig *tls.Config) ServerOption {
	return func(s *Server) {
		s.tlsConfig = tlsConfig
	}
}

func WithMaxRedirects(maxRedirects *int) ServerOption {
	return func(s *Server) {
		s.maxRedirects = maxRedirects
	}
}

func WithMasterName(masterName *string) ServerOption {
	return func(s *Server) {
		s.masterName = masterName
	}
}

func WithSentinelUsername(sentinelUsername *string) ServerOption {
	return func(s *Server) {
		s.sentinelUsername = sentinelUsername
	}
}

func WithSentinelPassword(sentinelPassword *string) ServerOption {
	return func(s *Server) {
		s.sentinelPassword = sentinelPassword
	}
}

func WithSentinelAuth(username, password *string) ServerOption {
	return func(s *Server) {
		s.sentinelUsername = username
		s.sentinelPassword = password
	}
}

func WithNetwork(network *string) ServerOption {
	return func(s *Server) {
		s.network = network
	}
}

func WithConcurrency(concurrency int) ServerOption {
	return func(s *Server) {
		s.asynqConfig.Concurrency = concurrency
	}
}

func WithConfig(cfg asynq.Config) ServerOption {
	return func(s *Server) {
		s.asynqConfig = cfg
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
