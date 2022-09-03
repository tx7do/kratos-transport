package machinery

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"sync"

	"github.com/RichardKnop/machinery/v2"
	"github.com/RichardKnop/machinery/v2/config"
	machineryLog "github.com/RichardKnop/machinery/v2/log"
	"github.com/RichardKnop/machinery/v2/tasks"

	redisBackend "github.com/RichardKnop/machinery/v2/backends/redis"
	redisBroker "github.com/RichardKnop/machinery/v2/brokers/redis"

	amqpBackend "github.com/RichardKnop/machinery/v2/backends/amqp"
	amqpBroker "github.com/RichardKnop/machinery/v2/brokers/amqp"

	eagerLock "github.com/RichardKnop/machinery/v2/locks/eager"

	ifaceBackends "github.com/RichardKnop/machinery/v2/backends/iface"
	ifaceBrokers "github.com/RichardKnop/machinery/v2/brokers/iface"
	ifaceLock "github.com/RichardKnop/machinery/v2/locks/iface"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
)

var (
	_ transport.Server     = (*Server)(nil)
	_ transport.Endpointer = (*Server)(nil)
)

const (
	defaultRedisAddress = "127.0.0.1:6379"
)

type redisOption struct {
	brokers  []string
	backends []string
	db       int
}

type consumerOption struct {
	consumerTag string // 消费者的标记
	concurrency int    // 并发数, 0表示不限制
}

type Server struct {
	sync.RWMutex
	started bool

	baseCtx context.Context
	err     error

	machineryServer *machinery.Server
	cfg             *config.Config

	redisOption    redisOption
	consumerOption consumerOption
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		baseCtx: context.Background(),
		started: false,
		cfg: &config.Config{
			DefaultQueue:    "kratos_tasks",
			ResultsExpireIn: 3600,
		},
		redisOption: redisOption{
			db: 0,
		},
		consumerOption: consumerOption{
			consumerTag: "machinery_worker",
			concurrency: 0,
		},
	}

	srv.init(opts...)

	return srv
}

func (s *Server) init(opts ...ServerOption) {
	for _, o := range opts {
		o(s)
	}

	if len(s.redisOption.brokers) > 0 && s.cfg.Broker == "" {
		s.cfg.Broker = s.redisOption.brokers[0]
	}
	if len(s.redisOption.backends) > 0 && s.cfg.ResultBackend == "" {
		s.cfg.ResultBackend = s.redisOption.backends[0]
	}

	s.initLogger()

	s.createMachineryServer()
}

func (s *Server) initLogger() {
	machineryLog.SetDebug(newLogger(log.LevelDebug))
	machineryLog.SetInfo(newLogger(log.LevelInfo))
	machineryLog.SetWarning(newLogger(log.LevelWarn))
	machineryLog.SetError(newLogger(log.LevelError))
	machineryLog.SetFatal(newLogger(log.LevelFatal))
}

func (s *Server) Name() string {
	return "machinery"
}

func (s *Server) Endpoint() (*url.URL, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.cfg == nil {
		return nil, nil
	}

	var addr string
	if len(s.redisOption.brokers) > 0 {
		addr = s.redisOption.brokers[0]
		if !strings.HasPrefix(addr, "redis://") {
			addr = "redis://" + addr
		}
	}

	return url.Parse(addr)
}

func (s *Server) HandleFunc(name string, handler interface{}) error {
	if err := s.registerTask(name, handler); err != nil {
		return err
	}
	return nil
}

// NewTask enqueue a new task
func (s *Server) NewTask(typeName string, values map[string]interface{}) error {
	signature := &tasks.Signature{
		Name: typeName,
	}

	for k, v := range values {
		signature.Args = append(signature.Args, tasks.Arg{Type: k, Value: v})
	}

	_, err := s.machineryServer.SendTask(signature)
	if err != nil {
		return err
	}
	return nil
}

// NewPeriodicTask 定时任务，不支持秒级任务
func (s *Server) NewPeriodicTask(spec, typeName string, values map[string]interface{}) error {
	signature := &tasks.Signature{
		Name: typeName,
	}

	for k, v := range values {
		signature.Args = append(signature.Args, tasks.Arg{Type: k, Value: v})
	}

	err := s.machineryServer.RegisterPeriodicTask(spec, typeName, signature)
	if err != nil {
		return err
	}

	_, err = s.machineryServer.SendTask(signature)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) Start(ctx context.Context) error {
	if s.err != nil {
		return s.err
	}

	if s.started {
		return nil
	}

	if err := s.newWorker(s.consumerOption.consumerTag, s.consumerOption.concurrency); err != nil {
		return err
	}

	endpoint, _ := s.Endpoint()
	log.Infof("[machinery] server listening on: %s", endpoint.String())

	s.baseCtx = ctx
	s.started = true

	return nil
}

func (s *Server) Stop(_ context.Context) error {
	log.Info("[machinery] server stopping")
	s.started = false

	s.machineryServer = nil

	return nil
}

func (s *Server) createMachineryServer() {
	var broker ifaceBrokers.Broker
	var backend ifaceBackends.Backend
	var lock ifaceLock.Lock

	if len(s.redisOption.brokers) > 0 && len(s.redisOption.backends) > 0 {
		broker = redisBroker.NewGR(s.cfg, s.redisOption.brokers, s.redisOption.db)
		backend = redisBackend.NewGR(s.cfg, s.redisOption.backends, s.redisOption.db)
	}
	if s.cfg.Redis != nil {
		broker = redisBroker.NewGR(s.cfg, []string{s.cfg.Broker}, s.redisOption.db)
		backend = redisBackend.NewGR(s.cfg, []string{s.cfg.ResultBackend}, s.redisOption.db)
	}

	if s.cfg.AMQP != nil {
		broker = amqpBroker.New(s.cfg)
		backend = amqpBackend.New(s.cfg)
	}

	lock = eagerLock.New()
	s.machineryServer = machinery.NewServer(s.cfg, broker, backend, lock)
}

func (s *Server) registerTask(name string, handler interface{}) error {
	if err := s.machineryServer.RegisterTask(name, handler); err != nil {
		return err
	}
	return nil
}

func (s *Server) newWorker(consumerTag string, concurrency int) error {
	worker := s.machineryServer.NewWorker(consumerTag, concurrency)
	if worker == nil {
		return errors.New("[machinery] create worker failed")
	}
	return worker.Launch()
}
