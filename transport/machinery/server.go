package machinery

import (
	"context"
	"errors"
	"net/url"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	semConv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"

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

	"github.com/tx7do/kratos-transport/tracing"
	"github.com/tx7do/kratos-transport/utils"
)

var (
	_ transport.Server     = (*Server)(nil)
	_ transport.Endpointer = (*Server)(nil)
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

	tracingOpts    []tracing.Option
	producerTracer *tracing.Tracer
	consumerTracer *tracing.Tracer

	keepAlive *utils.KeepAliveService
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

		keepAlive: utils.NewKeepAliveService(nil),
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

	if len(s.tracingOpts) > 0 {
		s.producerTracer = tracing.NewTracer(trace.SpanKindProducer, "machinery-producer", s.tracingOpts...)
		s.consumerTracer = tracing.NewTracer(trace.SpanKindConsumer, "machinery-consumer", s.tracingOpts...)
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

	return s.keepAlive.Endpoint()
}

func (s *Server) HandleFunc(name string, handler interface{}) error {
	if err := s.registerTask(name, handler); err != nil {
		return err
	}
	return nil
}

// NewTask enqueue a new task
func (s *Server) NewTask(typeName string, opts ...TaskOption) error {
	return s.newTask("", "", typeName, opts...)
}

// NewPeriodicTask 周期性定时任务，不支持秒级任务，最大精度只到分钟。
func (s *Server) NewPeriodicTask(cronSpec, typeName string, opts ...TaskOption) error {
	return s.newTask(cronSpec, typeName, typeName, opts...)
}

// NewGroup 执行一组异步任务，任务之间互不影响。
func (s *Server) NewGroup(groupTasks ...TasksOption) error {
	return s.newGroup("", "", 0, groupTasks...)
}
func (s *Server) NewPeriodicGroup(cronSpec string, groupTasks ...TasksOption) error {
	return s.newGroup(cronSpec, "periodic-group", 0, groupTasks...)
}

// NewChord 先执行一组同步任务，执行完成后，再调用最后一个回调函数。
func (s *Server) NewChord(chordTasks ...TasksOption) error {
	return s.newChord("", "", 0, chordTasks...)
}
func (s *Server) NewPeriodicChord(cronSpec string, chordTasks ...TasksOption) error {
	return s.newChord(cronSpec, "periodic-chord", 0, chordTasks...)
}

// NewChain 执行一组同步任务，任务有次序之分，上个任务的出参可作为下个任务的入参。
func (s *Server) NewChain(chainTasks ...TasksOption) error {
	return s.newChain("", "", chainTasks...)
}
func (s *Server) NewPeriodicChain(cronSpec string, chainTasks ...TasksOption) error {
	return s.newChain(cronSpec, "periodic-chain", chainTasks...)
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

	go func() {
		_ = s.keepAlive.Start()
	}()

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

	worker.SetPreTaskHandler(func(signature *tasks.Signature) {

	})

	return worker.Launch()
}

func (s *Server) newTask(cronSpec, lockName, typeName string, opts ...TaskOption) error {
	signature := &tasks.Signature{
		Name: typeName,
	}

	for _, o := range opts {
		o(signature)
	}

	var err error

	span := s.startProducerSpan(context.Background(), signature)
	defer s.finishProducerSpan(span, err)

	if len(cronSpec) > 0 {
		err = s.machineryServer.RegisterPeriodicTask(cronSpec, lockName, signature)
		if err != nil {
			return err
		}
	} else {
		_, err = s.machineryServer.SendTask(signature)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) newGroup(cronSpec, lockName string, concurrency int, groupTasks ...TasksOption) error {
	if len(groupTasks) == 0 {
		return errors.New("[machinery] group task is empty")
	}

	var signatures = make([]*tasks.Signature, 0, len(groupTasks))

	for _, o := range groupTasks {
		o(&signatures)
	}

	if len(signatures) == 0 {
		return errors.New("[machinery] group task is empty")
	}

	var err error

	if len(cronSpec) > 0 {
		if err := s.machineryServer.RegisterPeriodicGroup(cronSpec, lockName, concurrency, signatures...); err != nil {
			return err
		}
	} else {

		var group *tasks.Group
		group, err = tasks.NewGroup(signatures...)
		if err != nil {
			return err
		}

		_, err = s.machineryServer.SendGroup(group, concurrency)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) newChord(cronSpec, lockName string, concurrency int, groupTasks ...TasksOption) error {
	if len(groupTasks) < 2 {
		return errors.New("[machinery] chord task is empty")
	}

	var signatures = make([]*tasks.Signature, 0, len(groupTasks))

	for _, o := range groupTasks {
		o(&signatures)
	}

	var finalSignature *tasks.Signature
	finalSignature, signatures = signatures[len(signatures)-1], signatures[:len(signatures)-1]

	var err error

	if len(cronSpec) > 0 {
		if err := s.machineryServer.RegisterPeriodicChord(cronSpec, lockName, concurrency, finalSignature, signatures...); err != nil {
			return err
		}
	} else {
		var group *tasks.Group
		group, err = tasks.NewGroup(signatures...)
		if err != nil {
			return err
		}

		var chord *tasks.Chord
		chord, err = tasks.NewChord(group, finalSignature)
		if err != nil {
			return err
		}

		_, err = s.machineryServer.SendChord(chord, concurrency)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) newChain(cronSpec, lockName string, chainTasks ...TasksOption) error {
	if len(chainTasks) == 0 {
		return errors.New("[machinery] chain task is empty")
	}

	var signatures = make([]*tasks.Signature, 0, len(chainTasks))

	for _, o := range chainTasks {
		o(&signatures)
	}

	if len(signatures) == 0 {
		return errors.New("[machinery] chain task is empty")
	}

	var err error

	if len(cronSpec) > 0 {
		if err = s.machineryServer.RegisterPeriodicChain(cronSpec, lockName, signatures...); err != nil {
			return err
		}
	} else {
		var chain *tasks.Chain
		chain, err = tasks.NewChain(signatures...)
		if err != nil {
			return err
		}

		_, err = s.machineryServer.SendChain(chain)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) startProducerSpan(ctx context.Context, msg *tasks.Signature) trace.Span {
	if s.producerTracer == nil {
		return nil
	}

	carrier := NewMessageCarrier(&msg.Headers)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String("machinery"),
		semConv.MessagingDestinationKey.String(msg.Name),
	}

	var span trace.Span
	ctx, span = s.producerTracer.Start(ctx, carrier, attrs...)

	return span
}

func (s *Server) finishProducerSpan(span trace.Span, err error) {
	if s.producerTracer == nil {
		return
	}

	s.producerTracer.End(context.Background(), span, err)
}

func (s *Server) startConsumerSpan(ctx context.Context, msg *tasks.Signature) (context.Context, trace.Span) {
	if s.consumerTracer == nil {
		return ctx, nil
	}

	carrier := NewMessageCarrier(&msg.Headers)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String("machinery"),
		semConv.MessagingDestinationKindTopic,
		semConv.MessagingOperationReceive,
	}

	var span trace.Span
	ctx, span = s.consumerTracer.Start(ctx, carrier, attrs...)

	return ctx, span
}

func (s *Server) finishConsumerSpan(span trace.Span) {
	if s.consumerTracer == nil {
		return
	}

	s.consumerTracer.End(context.Background(), span, nil)
}
