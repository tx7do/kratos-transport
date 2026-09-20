package machinery

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semConv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/RichardKnop/machinery/v2"
	"github.com/RichardKnop/machinery/v2/config"
	machineryLog "github.com/RichardKnop/machinery/v2/log"
	"github.com/RichardKnop/machinery/v2/tasks"

	amqpBackend "github.com/RichardKnop/machinery/v2/backends/amqp"
	dynamoBackend "github.com/RichardKnop/machinery/v2/backends/dynamodb"
	eagerBackend "github.com/RichardKnop/machinery/v2/backends/eager"
	ifaceBackend "github.com/RichardKnop/machinery/v2/backends/iface"
	memcacheBackend "github.com/RichardKnop/machinery/v2/backends/memcache"
	mongoBackend "github.com/RichardKnop/machinery/v2/backends/mongo"
	redisBackend "github.com/RichardKnop/machinery/v2/backends/redis"

	amqpBroker "github.com/RichardKnop/machinery/v2/brokers/amqp"
	eagerBroker "github.com/RichardKnop/machinery/v2/brokers/eager"
	gcppubsubBroker "github.com/RichardKnop/machinery/v2/brokers/gcppubsub"
	ifaceBroker "github.com/RichardKnop/machinery/v2/brokers/iface"
	redisBroker "github.com/RichardKnop/machinery/v2/brokers/redis"
	sqsBroker "github.com/RichardKnop/machinery/v2/brokers/sqs"

	eagerLock "github.com/RichardKnop/machinery/v2/locks/eager"
	ifaceLock "github.com/RichardKnop/machinery/v2/locks/iface"
	redisLock "github.com/RichardKnop/machinery/v2/locks/redis"

	"github.com/go-kratos/kratos/v2/log"
	kratosTransport "github.com/go-kratos/kratos/v2/transport"

	"github.com/tx7do/kratos-transport/tracing"
	"github.com/tx7do/kratos-transport/transport/keepalive"
)

var (
	_ kratosTransport.Server     = (*Server)(nil)
	_ kratosTransport.Endpointer = (*Server)(nil)
)

const (
	TracerMessageSystemKey = "machinery"
	SpanNameProducer       = "machinery-producer"
	SpanNameConsumer       = "machinery-consumer"
)

type Server struct {
	sync.RWMutex
	started atomic.Bool

	baseCtx context.Context
	err     error

	machineryServer *machinery.Server
	worker          *machinery.Worker
	workerErrChan   chan error

	cfg *config.Config

	brokerOption   brokerOption
	backendOption  backendOption
	lockOption     lockOption
	consumerOption consumerOption

	tracingOpts    []tracing.Option
	producerTracer *tracing.Tracer
	consumerTracer *tracing.Tracer

	keepaliveServer *keepalive.Server
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		baseCtx: context.Background(),
		started: atomic.Bool{},

		cfg: &config.Config{
			DefaultQueue:    "kratos_machinery_queue",
			ResultsExpireIn: 3600,

			// 进程信号由 kratos 应用统一处理，worker 不自行捕获 SIGINT/SIGTERM
			NoUnixSignals: true,

			AMQP: &config.AMQPConfig{},
			SQS:  &config.SQSConfig{},
			Redis: &config.RedisConfig{
				MaxIdle:                3,
				IdleTimeout:            240,
				ReadTimeout:            15,
				WriteTimeout:           15,
				ConnectTimeout:         15,
				NormalTasksPollPeriod:  1000,
				DelayedTasksPollPeriod: 500,
			},
			GCPPubSub: &config.GCPPubSubConfig{},
			MongoDB:   &config.MongoDBConfig{},
			DynamoDB:  &config.DynamoDBConfig{},
		},
		consumerOption: consumerOption{
			consumerTag: "kratos_machinery_worker",
			concurrency: 1,
			queue:       "kratos_machinery_queue",
		},
		brokerOption: brokerOption{
			brokerType: BrokerTypeRedis,
			db:         0,
		},
		backendOption: backendOption{
			backendType: BackendTypeRedis,
			db:          0,
		},
		lockOption: lockOption{
			lockType: LockTypeRedis,
			db:       0,
			retries:  1,
		},
	}

	srv.init(opts...)

	return srv
}

func (s *Server) init(opts ...ServerOption) {
	for _, o := range opts {
		o(s)
	}

	if len(s.tracingOpts) > 0 {
		s.producerTracer = tracing.NewTracer(trace.SpanKindProducer, SpanNameProducer, s.tracingOpts...)
		s.consumerTracer = tracing.NewTracer(trace.SpanKindConsumer, SpanNameConsumer, s.tracingOpts...)
	}

	s.keepaliveServer = keepalive.NewServer(
		keepalive.WithServiceKind(KindMachinery),
	)

	s.installLogger()

	if err := s.createMachineryServer(); err != nil {
		s.err = err
	}
}

func (s *Server) Name() string {
	return KindMachinery
}

func (s *Server) HandleFunc(name string, handler any) error {
	if err := s.registerTask(name, handler); err != nil {
		return err
	}
	return nil
}

// NewTask enqueue a new task
func (s *Server) NewTask(ctx context.Context, typeName string, opts ...TaskOption) error {
	return s.newTask(ctx, "", "", typeName, opts...)
}

// NewPeriodicTask 周期性定时任务，不支持秒级任务，最大精度只到分钟。
func (s *Server) NewPeriodicTask(ctx context.Context, cronSpec, typeName string, opts ...TaskOption) error {
	return s.newTask(ctx, cronSpec, typeName, typeName, opts...)
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

	if s.started.Load() {
		return nil
	}

	if s.keepaliveServer == nil {
		// Stop 置 nil 后重启时重建，否则 keepalive/Endpoint 永久失效
		s.keepaliveServer = keepalive.NewServer(keepalive.WithServiceKind(KindMachinery))
	}

	if err := s.startWorker(
		s.consumerOption.consumerTag,
		s.consumerOption.concurrency,
		s.consumerOption.queue,
	); err != nil {
		return err
	}

	LogInfof("server started")

	// worker 启动成功后再起 keepalive，失败路径上不会泄漏 goroutine；
	// 先捕获局部变量：Stop 会把字段置 nil，goroutine 内再解引用字段会 nil panic
	if ka := s.keepaliveServer; ka != nil {
		go func() {
			if err := ka.Start(ctx); err != nil {
				LogErrorf("keepalive server start failed: %s", err.Error())
			}
		}()
	}

	s.baseCtx = ctx
	s.started.Store(true)

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	LogInfo("server stopping...")

	s.started.Store(false)

	if s.worker != nil {
		// 通知 worker 退出消费循环
		s.worker.Quit()

		// 等待 worker 消费循环收敛，超时则放弃等待
		if s.workerErrChan != nil {
			select {
			case err := <-s.workerErrChan:
				if err != nil && !errors.Is(err, machinery.ErrWorkerQuitGracefully) {
					LogErrorf("worker exited with error: %s", err.Error())
				}
			case <-ctx.Done():
				LogWarn("wait worker stop timeout")
			case <-time.After(10 * time.Second):
				LogWarn("wait worker stop timeout")
			}
		}
		s.worker = nil
		s.workerErrChan = nil
	}

	if s.keepaliveServer != nil {
		if err := s.keepaliveServer.Stop(ctx); err != nil {
			LogError("keepalive server stop failed", err)
		}
		s.keepaliveServer = nil
	}

	s.err = nil

	LogInfo("server stopped.")

	return nil
}

// installLogger 安装日志记录器
func (s *Server) installLogger() {
	machineryLog.SetDebug(newLogger(log.LevelDebug))
	machineryLog.SetInfo(newLogger(log.LevelInfo))
	machineryLog.SetWarning(newLogger(log.LevelWarn))
	machineryLog.SetError(newLogger(log.LevelError))
	machineryLog.SetFatal(newLogger(log.LevelFatal))
}

func (s *Server) createMachineryServer() error {
	var broker ifaceBroker.Broker
	var backend ifaceBackend.Backend
	var lock ifaceLock.Lock

	var err error

	if s.cfg.Broker != "" {
		switch s.brokerOption.brokerType {
		case BrokerTypeRedis:
			broker = redisBroker.NewGR(s.cfg, []string{s.cfg.Broker}, s.brokerOption.db)
			break
		case BrokerTypeAmqp:
			broker = amqpBroker.New(s.cfg)
			break
		case BrokerTypeGcpPubSub:
			if broker, err = gcppubsubBroker.New(s.cfg, s.brokerOption.projectID, s.brokerOption.subscriptionName); err != nil {
				LogError("create GCP PubSub broker error:", err)
			}
			break
		case BrokerTypeSQS:
			if broker, err = sqsBroker.New(s.cfg); err != nil {
				LogError("create SQS broker error:", err)
			}
			break
		}
	}

	if s.cfg.ResultBackend != "" {
		switch s.backendOption.backendType {
		case BackendTypeRedis:
			backend = redisBackend.NewGR(s.cfg, []string{s.cfg.ResultBackend}, s.backendOption.db)
			break
		case BackendTypeAmqp:
			backend = amqpBackend.New(s.cfg)
			break
		case BackendTypeMemcache:
			backend = memcacheBackend.New(s.cfg, []string{s.cfg.ResultBackend})
			break
		case BackendTypeMongoDB:
			if backend, err = mongoBackend.New(s.cfg); err != nil {
				LogError("create MongoDB backend error:", err)
			}
			break
		case BackendTypeDynamoDB:
			if backend, err = dynamoBackend.New(s.cfg); err != nil {
				LogError("create DynamoDB backend error:", err)
			}
			break
		}
	}

	if s.cfg.Lock != "" {
		switch s.lockOption.lockType {
		case LockTypeRedis:
			lock = redisLock.New(s.cfg, []string{s.cfg.Lock}, s.lockOption.db, s.lockOption.retries)
			break
		}
	}

	// 用户显式配置了 broker 但创建失败时，不应静默降级为 eager
	// （eager = 进程内同步执行，任务零持久化零重试，机器故障即任务丢失）
	if s.cfg.Broker != "" && broker == nil {
		return fmt.Errorf("broker address %q is configured but no broker was created (check brokerType)", s.cfg.Broker)
	}

	if broker == nil {
		broker = eagerBroker.New()
	}
	if backend == nil {
		backend = eagerBackend.New()
	}
	if lock == nil {
		lock = eagerLock.New()
	}

	s.machineryServer = machinery.NewServer(s.cfg, broker, backend, lock)
	return nil
}

func (s *Server) registerTask(name string, handler any) error {
	if err := s.machineryServer.RegisterTask(name, handler); err != nil {
		return err
	}
	return nil
}

// startWorker 以非阻塞方式启动 worker。
// machinery 的 worker.Launch() 会阻塞到 worker 退出，不能在 Start 里同步调用，
// 否则 Start 永不返回。
func (s *Server) startWorker(consumerTag string, concurrency int, queue string) error {
	if s.machineryServer == nil {
		return errors.New("machinery server is nil")
	}

	worker := s.machineryServer.NewCustomQueueWorker(consumerTag, concurrency, queue)
	if worker == nil {
		return errors.New("create worker failed")
	}

	worker.SetPreTaskHandler(func(signature *tasks.Signature) {

	})

	// worker 退出时向 errChan 发送最终错误；cap 2 兜底 NoUnixSignals=false 时
	// 信号路径可能的额外发送，避免发送 goroutine 永久阻塞
	errChan := make(chan error, 2)

	worker.LaunchAsync(errChan)

	s.worker = worker
	s.workerErrChan = errChan

	return nil
}

func (s *Server) newTask(ctx context.Context, cronSpec, lockName, typeName string, opts ...TaskOption) error {
	signature := &tasks.Signature{
		Name: typeName,
	}

	for _, o := range opts {
		o(signature)
	}

	var err error

	var span trace.Span
	// 使用调用方 ctx：保留取消语义与 tracing 关联
	ctx, span = s.startProducerSpan(ctx, signature)
	defer func() { s.finishProducerSpan(ctx, span, err) }()

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
		return errors.New("group task is empty")
	}

	var signatures = make([]*tasks.Signature, 0, len(groupTasks))

	for _, o := range groupTasks {
		o(&signatures)
	}

	if len(signatures) == 0 {
		return errors.New("group task is empty")
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
		return errors.New("chord task is empty")
	}

	var signatures = make([]*tasks.Signature, 0, len(groupTasks))

	for _, o := range groupTasks {
		o(&signatures)
	}

	var finalSignature *tasks.Signature
	finalSignature, signatures = signatures[len(signatures)-1], signatures[:len(signatures)-1]

	var err error

	if len(cronSpec) > 0 {
		if err = s.machineryServer.RegisterPeriodicChord(cronSpec, lockName, concurrency, finalSignature, signatures...); err != nil {
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
		return errors.New("chain task is empty")
	}

	var signatures = make([]*tasks.Signature, 0, len(chainTasks))

	for _, o := range chainTasks {
		o(&signatures)
	}

	if len(signatures) == 0 {
		return errors.New("chain task is empty")
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

func (s *Server) startProducerSpan(ctx context.Context, msg *tasks.Signature) (context.Context, trace.Span) {
	if s.producerTracer == nil {
		return ctx, nil
	}

	if msg == nil {
		return ctx, nil
	}

	carrier := NewMessageCarrier(&msg.Headers)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String(TracerMessageSystemKey),
		semConv.MessagingDestinationKey.String(msg.Name),
	}

	var span trace.Span
	ctx, span = s.producerTracer.Start(ctx, carrier, attrs...)

	if span != nil {
		otel.GetTextMapPropagator().Inject(ctx, carrier)
	}

	return ctx, span
}

func (s *Server) finishProducerSpan(ctx context.Context, span trace.Span, err error) {
	if s.producerTracer == nil {
		return
	}

	s.producerTracer.End(ctx, span, err)
}

func (s *Server) startConsumerSpan(ctx context.Context, msg *tasks.Signature) (context.Context, trace.Span) {
	if s.consumerTracer == nil {
		return ctx, nil
	}

	carrier := NewMessageCarrier(&msg.Headers)

	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String(TracerMessageSystemKey),
		semConv.MessagingDestinationKindTopic,
		semConv.MessagingOperationReceive,
	}

	var span trace.Span
	ctx, span = s.consumerTracer.Start(ctx, carrier, attrs...)

	return ctx, span
}

func (s *Server) finishConsumerSpan(ctx context.Context, span trace.Span, err error) {
	if s.consumerTracer == nil {
		return
	}

	s.consumerTracer.End(ctx, span, err)
}

func (s *Server) Endpoint() (*url.URL, error) {
	if s.keepaliveServer == nil {
		return nil, errors.New("keepalive server is nil")
	}

	return s.keepaliveServer.Endpoint()
}
