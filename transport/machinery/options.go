package machinery

import (
	"crypto/tls"
	"time"

	"github.com/RichardKnop/machinery/v2/config"
	"github.com/RichardKnop/machinery/v2/tasks"
)

type BrokerType int
type BackendType int
type LockType int

const (
	BrokerTypeRedis     BrokerType = iota // Redis
	BrokerTypeAmqp                        // AMQP
	BrokerTypeGcpPubSub                   // GCP Pub/Sub
	BrokerTypeSQS                         // AWS SQS
)

const (
	BackendTypeRedis    BackendType = iota // Redis
	BackendTypeAmqp                        // AMQP
	BackendTypeMemcache                    // Memcache
	BackendTypeMongoDB                     // MongoDB
	BackendTypeDynamoDB                    // Amazon DynamoDB
)

const (
	LockTypeRedis LockType = iota
)

type brokerOption struct {
	brokerType BrokerType

	db int

	projectID, subscriptionName string
}

type backendOption struct {
	backendType BackendType

	db int
}

type lockOption struct {
	lockType LockType

	db      int
	retries int
}

type consumerOption struct {
	consumerTag string // 消费者的标记
	concurrency int    // 并发数, 0表示不限制
	queue       string
}

//////////////////////////////////////////////////////////////////////////////////////////////////

type ServerOption func(o *Server)

// WithYamlConfig read config from yaml file.
func WithYamlConfig(cnfPath string, keepReloading bool) ServerOption {
	return func(s *Server) {
		cnf, err := config.NewFromYaml(cnfPath, keepReloading)
		if err != nil {
			LogErrorf("load yaml config [%s] failed: %s", cnfPath, err.Error())
		}
		s.cfg = cnf
	}
}

// WithEnvironmentConfig read config from env.
func WithEnvironmentConfig() ServerOption {
	return func(s *Server) {
		cnf, err := config.NewFromEnvironment()
		if err != nil {
			LogErrorf("load environment config failed: %s", err.Error())
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

// WithEnableKeepAlive enable keep alive
func WithEnableKeepAlive(enable bool) ServerOption {
	return func(s *Server) {
		s.enableKeepAlive = enable
	}
}

func WithBrokerAddress(addr string, db int, brokerType BrokerType) ServerOption {
	return func(s *Server) {
		s.cfg.Broker = addr
		s.brokerOption.db = db
		s.brokerOption.brokerType = brokerType
	}
}

func WithResultBackendAddress(addr string, db int, backendType BackendType) ServerOption {
	return func(s *Server) {
		s.cfg.ResultBackend = addr
		s.backendOption.db = db
		s.backendOption.backendType = backendType
	}
}

func WithLockAddress(addr string, db, retries int, lockType LockType) ServerOption {
	return func(s *Server) {
		s.cfg.Lock = addr
		s.lockOption.db = db
		s.lockOption.retries = retries
		s.lockOption.lockType = lockType
	}
}

func WithRedisConfig(cfg *config.RedisConfig) ServerOption {
	return func(s *Server) {
		s.cfg.Redis = cfg
	}
}

func WithAMQPConfig(cfg *config.AMQPConfig) ServerOption {
	return func(s *Server) {
		s.cfg.AMQP = cfg
	}
}

func WithSQSConfig(cfg *config.SQSConfig) ServerOption {
	return func(s *Server) {
		s.cfg.SQS = cfg
	}
}

func WithGCPPubSubConfig(cfg *config.GCPPubSubConfig) ServerOption {
	return func(s *Server) {
		s.cfg.GCPPubSub = cfg
	}
}

func WithMongoDBConfig(cfg *config.MongoDBConfig) ServerOption {
	return func(s *Server) {
		s.cfg.MongoDB = cfg
	}
}

func WithDynamoDBConfig(cfg *config.DynamoDBConfig) ServerOption {
	return func(s *Server) {
		s.cfg.DynamoDB = cfg
	}
}

func WithConsumerOption(consumerTag string, concurrency int, queue string) ServerOption {
	return func(s *Server) {
		s.consumerOption.consumerTag = consumerTag
		s.consumerOption.concurrency = concurrency
		s.consumerOption.queue = queue
	}
}

func WithDefaultQueue(name string) ServerOption {
	return func(s *Server) {
		s.cfg.DefaultQueue = name
	}
}

func WithResultsExpireIn(expire int) ServerOption {
	return func(s *Server) {
		s.cfg.ResultsExpireIn = expire
	}
}

func WithNoUnixSignals(noUnixSignals bool) ServerOption {
	return func(s *Server) {
		s.cfg.NoUnixSignals = noUnixSignals
	}
}

//////////////////////////////////////////////////////////////////////////////////////////////////

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

//////////////////////////////////////////////////////////////////////////////////////////////////

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
