package asynq

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hibiken/asynq"

	"github.com/go-kratos/kratos/v2/encoding"
	kratosTransport "github.com/go-kratos/kratos/v2/transport"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/transport/keepalive"
)

var (
	_ kratosTransport.Server     = (*Server)(nil)
	_ kratosTransport.Endpointer = (*Server)(nil)
)

type Server struct {
	sync.RWMutex

	started atomic.Bool

	baseCtx context.Context
	err     error

	server *asynq.Server
	client *asynq.Client

	scheduler *asynq.Scheduler
	inspector *asynq.Inspector

	mux           *asynq.ServeMux
	asynqConfig   asynq.Config
	redisConnOpt  asynq.RedisConnOpt
	schedulerOpts *asynq.SchedulerOpts

	addresses        []string
	username         *string
	password         *string
	db               *int
	poolSize         *int
	dialTimeout      *time.Duration
	readTimeout      *time.Duration
	writeTimeout     *time.Duration
	tlsConfig        *tls.Config
	maxRedirects     *int
	masterName       *string
	sentinelUsername *string
	sentinelPassword *string
	network          *string

	gracefullyShutdown bool

	codec encoding.Codec

	entryIDs    map[string]string
	mtxEntryIDs sync.RWMutex

	typeNameMap sync.Map

	keepaliveServer *keepalive.Server
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		baseCtx:      context.Background(),
		started:      atomic.Bool{},
		redisConnOpt: newRedisClientOpt(),
		asynqConfig: asynq.Config{
			Concurrency: defaultConcurrency,
			Logger:      newLogger(),
		},
		schedulerOpts: &asynq.SchedulerOpts{},
		mux:           asynq.NewServeMux(),

		codec: encoding.GetCodec("json"),

		entryIDs:    make(map[string]string),
		mtxEntryIDs: sync.RWMutex{},

		typeNameMap: sync.Map{},

		gracefullyShutdown: false,
	}

	srv.init(opts...)

	return srv
}

func (s *Server) updateRedisClientOpt(opt *asynq.RedisClientOpt) {
	if s.username != nil {
		opt.Username = *s.username
	}
	if s.password != nil {
		opt.Password = *s.password
	}
	if s.db != nil {
		opt.DB = *s.db
	}
	if len(s.addresses) > 0 {
		opt.Addr = s.addresses[0]
	}
	if s.poolSize != nil {
		opt.PoolSize = *s.poolSize
	}
	if s.dialTimeout != nil {
		opt.DialTimeout = *s.dialTimeout
	}
	if s.readTimeout != nil {
		opt.ReadTimeout = *s.readTimeout
	}
	if s.writeTimeout != nil {
		opt.WriteTimeout = *s.writeTimeout
	}
	if s.tlsConfig != nil {
		opt.TLSConfig = s.tlsConfig
	}
	if s.network != nil {
		opt.Network = *s.network
	}
}

func (s *Server) updateRedisClusterClientOpt(opt *asynq.RedisClusterClientOpt) {
	if s.username != nil {
		opt.Username = *s.username
	}
	if s.password != nil {
		opt.Password = *s.password
	}
	if len(s.addresses) > 0 {
		opt.Addrs = s.addresses
	}
	if s.dialTimeout != nil {
		opt.DialTimeout = *s.dialTimeout
	}
	if s.readTimeout != nil {
		opt.ReadTimeout = *s.readTimeout
	}
	if s.writeTimeout != nil {
		opt.WriteTimeout = *s.writeTimeout
	}
	if s.tlsConfig != nil {
		opt.TLSConfig = s.tlsConfig
	}
	if s.maxRedirects != nil {
		opt.MaxRedirects = *s.maxRedirects
	}
}

func (s *Server) updateRedisFailoverClientOpt(opt *asynq.RedisFailoverClientOpt) {
	if s.username != nil {
		opt.Username = *s.username
	}
	if s.password != nil {
		opt.Password = *s.password
	}
	if s.db != nil {
		opt.DB = *s.db
	}
	if len(s.addresses) > 0 {
		opt.SentinelAddrs = s.addresses
	}
	if s.poolSize != nil {
		opt.PoolSize = *s.poolSize
	}
	if s.dialTimeout != nil {
		opt.DialTimeout = *s.dialTimeout
	}
	if s.readTimeout != nil {
		opt.ReadTimeout = *s.readTimeout
	}
	if s.writeTimeout != nil {
		opt.WriteTimeout = *s.writeTimeout
	}
	if s.tlsConfig != nil {
		opt.TLSConfig = s.tlsConfig
	}
	if s.masterName != nil {
		opt.MasterName = *s.masterName
	}
	if s.sentinelUsername != nil {
		opt.SentinelUsername = *s.sentinelUsername
	}
	if s.sentinelPassword != nil {
		opt.SentinelPassword = *s.sentinelPassword
	}
}

func (s *Server) init(opts ...ServerOption) {
	for _, o := range opts {
		o(s)
	}

	switch v := s.redisConnOpt.(type) {
	case *asynq.RedisClientOpt:
		s.updateRedisClientOpt(v)
	case asynq.RedisClientOpt:
		s.updateRedisClientOpt(&v)

	case *asynq.RedisClusterClientOpt:
		s.updateRedisClusterClientOpt(v)
	case asynq.RedisClusterClientOpt:
		s.updateRedisClusterClientOpt(&v)

	case *asynq.RedisFailoverClientOpt:
		s.updateRedisFailoverClientOpt(v)
	case asynq.RedisFailoverClientOpt:
		s.updateRedisFailoverClientOpt(&v)
	}

	s.keepaliveServer = keepalive.NewServer(
		keepalive.WithServiceKind(KindAsynq),
	)

	var err error
	if err = s.createAsynqServer(); err != nil {
		s.err = err
		LogError("create asynq server failed:", err)
	}
	if err = s.createAsynqClient(); err != nil {
		s.err = err
		LogError("create asynq client failed:", err)
	}
	if err = s.createAsynqScheduler(); err != nil {
		s.err = err
		LogError("create asynq scheduler failed:", err)
	}
	if err = s.createAsynqInspector(); err != nil {
		s.err = err
		LogError("create asynq inspector failed:", err)
	}
}

// Name returns the name of server
func (s *Server) Name() string {
	return KindAsynq
}

// TaskTypeExists check if task type is registered
func (s *Server) TaskTypeExists(taskType string) bool {
	_, ok := s.typeNameMap.Load(taskType)
	return ok
}

// GetRegisteredTaskTypes get all registered task types
func (s *Server) GetRegisteredTaskTypes() []string {
	var types []string
	s.typeNameMap.Range(func(key, value any) bool {
		if typeName, ok := key.(string); ok {
			types = append(types, typeName)
		}
		return true
	})
	return types
}

// RegisterSubscriber register task subscriber
func (s *Server) RegisterSubscriber(taskType string, handler MessageHandler, creator Creator) error {
	err := s.handleFunc(taskType, func(ctx context.Context, task *asynq.Task) error {
		var payload MessagePayload

		if creator != nil {
			payload = creator()

			if err := broker.Unmarshal(s.codec, task.Payload(), &payload); err != nil {
				LogErrorf("unmarshal message failed: %s", err)
				return err
			}
		} else {
			payload = task.Payload()
		}

		if err := handler(task.Type(), payload); err != nil {
			LogErrorf("handle message failed: %s", err)
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	s.typeNameMap.Store(taskType, true)

	return nil
}

// RegisterSubscriber register task subscriber
func RegisterSubscriber[T any](srv *Server, taskType string, handler func(string, *T) error) error {
	return srv.RegisterSubscriber(taskType,
		func(taskType string, payload MessagePayload) error {
			switch t := payload.(type) {
			case *T:
				return handler(taskType, t)
			default:
				LogError("invalid payload struct type:", t)
				return errors.New("invalid payload struct type")
			}
		},
		func() any {
			var t T
			return &t
		},
	)
}

// RegisterSubscriberWithCtx register task subscriber with context
func (s *Server) RegisterSubscriberWithCtx(
	taskType string,
	handler func(context.Context, string, MessagePayload) error,
	creator Creator,
) error {
	err := s.handleFunc(taskType, func(ctx context.Context, task *asynq.Task) error {
		var payload MessagePayload
		if creator != nil {
			payload = creator()

			if err := broker.Unmarshal(s.codec, task.Payload(), &payload); err != nil {
				LogErrorf("unmarshal message failed: %s", err)
				return err
			}
		} else {
			payload = task.Payload()
		}

		if err := handler(ctx, task.Type(), payload); err != nil {
			LogErrorf("handle message failed: %s", err)
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	s.typeNameMap.Store(taskType, true)

	return nil
}

// RegisterSubscriberWithCtx register task subscriber with context
func RegisterSubscriberWithCtx[T any](srv *Server, taskType string,
	handler func(context.Context, string, *T) error) error {
	return srv.RegisterSubscriberWithCtx(taskType,
		func(ctx context.Context, taskType string, payload MessagePayload) error {
			switch t := payload.(type) {
			case *T:
				return handler(ctx, taskType, t)
			default:
				LogError("invalid payload struct type:", t)
				return errors.New("invalid payload struct type")
			}
		},
		func() any {
			var t T
			return &t
		},
	)
}

func (s *Server) handleFunc(pattern string, handler func(context.Context, *asynq.Task) error) error {
	if s.started.Load() {
		LogErrorf("handleFunc [%s] failed", pattern)
		return errors.New("cannot handle func, server already started")
	}
	s.mux.HandleFunc(pattern, handler)
	return nil
}

// NewTask enqueue a new task
func (s *Server) NewTask(typeName string, msg broker.Any, opts ...asynq.Option) error {
	//if !s.started.Load() {
	//	return errors.New("cannot create task, server already started")
	//}

	if typeName == "" {
		return errors.New("typeName cannot be empty")
	}

	if s.client == nil {
		if err := s.createAsynqClient(); err != nil {
			return err
		}
	}

	var err error

	var payload []byte
	if payload, err = broker.Marshal(s.codec, msg); err != nil {
		return err
	}

	task := asynq.NewTask(typeName, payload, opts...)
	if task == nil {
		return errors.New("new task failed")
	}

	taskInfo, err := s.client.Enqueue(task, opts...)
	if err != nil {
		LogErrorf("[%s] Enqueue failed: %s", typeName, err.Error())
		return err
	}

	LogDebugf("[%s] enqueued task: id=%s queue=%s", typeName, taskInfo.ID, taskInfo.Queue)

	return nil
}

// NewWaitResultTask enqueue a new task and wait for the result
func (s *Server) NewWaitResultTask(typeName string, msg broker.Any, opts ...asynq.Option) error {
	//if !s.started.Load() {
	//	return errors.New("cannot create task, server already started")
	//}

	if typeName == "" {
		return errors.New("typeName cannot be empty")
	}

	if s.client == nil {
		if err := s.createAsynqClient(); err != nil {
			return err
		}
	}

	var err error

	var payload []byte
	if payload, err = broker.Marshal(s.codec, msg); err != nil {
		return err
	}

	task := asynq.NewTask(typeName, payload, opts...)
	if task == nil {
		return errors.New("new task failed")
	}

	taskInfo, err := s.client.Enqueue(task, opts...)
	if err != nil {
		LogErrorf("[%s] Enqueue failed: %s", typeName, err.Error())
		return err
	}

	if s.inspector == nil {
		if err = s.createAsynqInspector(); err != nil {
			return err
		}
	}

	_, err = waitResult(s.inspector, taskInfo)
	if err != nil {
		LogErrorf("[%s] wait result failed: %s", typeName, err.Error())
		return err
	}

	LogDebugf("[%s] enqueued task: id=%s queue=%s", typeName, taskInfo.ID, taskInfo.Queue)

	return nil
}

func waitResult(intor *asynq.Inspector, info *asynq.TaskInfo) (*asynq.TaskInfo, error) {
	taskInfo, err := intor.GetTaskInfo(info.Queue, info.ID)
	if err != nil {
		return nil, err
	}

	if taskInfo.State != asynq.TaskStateCompleted && taskInfo.State != asynq.TaskStateArchived && taskInfo.State != asynq.TaskStateRetry {
		return waitResult(intor, info)
	}

	if taskInfo.State == asynq.TaskStateRetry {
		return nil, fmt.Errorf("task state is %s", taskInfo.State.String())
	}

	return taskInfo, nil
}

// NewPeriodicTask enqueue a new crontab task
func (s *Server) NewPeriodicTask(cronSpec, typeName string, msg broker.Any, opts ...asynq.Option) (string, error) {
	//if !s.started.Load() {
	//	return "", errors.New("cannot create periodic task, server already started")
	//}

	if cronSpec == "" {
		return "", errors.New("cronSpec cannot be empty")
	}
	if typeName == "" {
		return "", errors.New("typeName cannot be empty")
	}

	if s.scheduler == nil {
		if err := s.createAsynqScheduler(); err != nil {
			return "", err
		}
		if err := s.runAsynqScheduler(); err != nil {
			return "", err
		}
	}

	payload, err := broker.Marshal(s.codec, msg)
	if err != nil {
		return "", err
	}

	var options []asynq.Option
	if len(opts) > 0 {
		options = opts
	} else {
		options = []asynq.Option{}
	}

	task := asynq.NewTask(typeName, payload, options...)
	if task == nil {
		return "", errors.New("new task failed")
	}

	entryID, err := s.scheduler.Register(cronSpec, task, opts...)
	if err != nil {
		LogErrorf("[%s] enqueue periodic task failed: %s", typeName, err.Error())
		return "", err
	}

	s.addPeriodicTaskEntryID(typeName, entryID)

	LogDebugf("[%s]  registered an entry: id=%q", typeName, entryID)

	return entryID, nil
}

// RemovePeriodicTask remove periodic task
func (s *Server) RemovePeriodicTask(taskId string) error {
	entryId := s.QueryPeriodicTaskEntryID(taskId)
	if entryId == "" {
		return errors.New(fmt.Sprintf("[%s] periodic task not exist", taskId))
	}

	if err := s.unregisterPeriodicTask(entryId); err != nil {
		LogErrorf("[%s] dequeue periodic task failed: %s", entryId, err.Error())
		return err
	}

	s.removePeriodicTaskEntryID(entryId)

	return nil
}

func (s *Server) RemoveAllPeriodicTask() {
	s.mtxEntryIDs.Lock()
	ids := s.entryIDs
	s.entryIDs = make(map[string]string)
	s.mtxEntryIDs.Unlock()

	for _, v := range ids {
		_ = s.unregisterPeriodicTask(v)
	}
}

func (s *Server) unregisterPeriodicTask(entryId string) error {
	if s.scheduler == nil {
		return nil
	}

	if err := s.scheduler.Unregister(entryId); err != nil {
		LogErrorf("[%s] dequeue periodic task failed: %s", entryId, err.Error())
		return err
	}

	return nil
}

func (s *Server) addPeriodicTaskEntryID(taskId, entryId string) {
	s.mtxEntryIDs.Lock()
	defer s.mtxEntryIDs.Unlock()

	s.entryIDs[taskId] = entryId
}

func (s *Server) removePeriodicTaskEntryID(taskId string) {
	s.mtxEntryIDs.Lock()
	defer s.mtxEntryIDs.Unlock()

	delete(s.entryIDs, taskId)
}

func (s *Server) QueryPeriodicTaskEntryID(taskId string) string {
	s.mtxEntryIDs.RLock()
	defer s.mtxEntryIDs.RUnlock()

	entryID, ok := s.entryIDs[taskId]
	if !ok {
		return ""
	}
	return entryID
}

// Start the server
func (s *Server) Start(ctx context.Context) error {
	if s.err != nil {
		return s.err
	}

	if s.started.Load() {
		return nil
	}

	if s.keepaliveServer != nil {
		go func() {
			if s.err = s.keepaliveServer.Start(ctx); s.err != nil {
				LogErrorf("keepalive server start failed: %s", s.err.Error())
			}
		}()
	}

	if s.err = s.runAsynqScheduler(); s.err != nil {
		LogError("run asynq scheduler failed", s.err)
		return s.err
	}

	if s.err = s.runAsynqServer(); s.err != nil {
		LogError("run asynq server failed", s.err)
		return s.err
	}

	s.baseCtx = ctx
	s.started.Store(true)

	return nil
}

// Stop the server
func (s *Server) Stop(ctx context.Context) error {
	//if !s.started.Load() {
	//	return nil
	//}

	LogInfo("server stopping...")

	s.started.Store(false)

	if s.client != nil {
		_ = s.client.Close()
		s.client = nil
	}

	if s.server != nil {
		if s.gracefullyShutdown {
			LogInfo("server gracefully shutdown")
			s.server.Shutdown()
		} else {
			s.server.Stop()
		}
		s.server = nil
	}

	if s.scheduler != nil {
		s.scheduler.Shutdown()
		s.scheduler = nil
	}

	if s.inspector != nil {
		_ = s.inspector.Close()
		s.inspector = nil
	}
	s.err = nil

	if s.keepaliveServer != nil {
		if err := s.keepaliveServer.Stop(ctx); err != nil {
			LogError("keepalive server stop failed", s.err)
		}
		s.keepaliveServer = nil
	}

	LogInfo("server stopped.")

	return nil
}

// createAsynqServer create asynq server
func (s *Server) createAsynqServer() error {
	if s.server != nil {
		return nil
	}

	s.server = asynq.NewServer(s.redisConnOpt, s.asynqConfig)
	if s.server == nil {
		LogErrorf("create asynq server failed")
		return errors.New("create asynq server failed")
	}
	return nil
}

// runAsynqServer run asynq server
func (s *Server) runAsynqServer() error {
	if s.server == nil {
		LogErrorf("asynq server is nil")
		return errors.New("asynq server is nil")
	}

	go func() {
		if s.server == nil {
			s.err = errors.New("asynq server is nil")
			LogErrorf("asynq server run failed: %s", s.err.Error())
			return
		}

		if s.err = s.server.Run(s.mux); s.err != nil {
			LogErrorf("asynq server run failed: %s", s.err.Error())
			return
		}
	}()

	LogInfo("asynq server started")

	return nil
}

// createAsynqClient create asynq client
func (s *Server) createAsynqClient() error {
	if s.client != nil {
		return nil
	}

	s.client = asynq.NewClient(s.redisConnOpt)
	if s.client == nil {
		LogErrorf("create asynq client failed")
		return errors.New("create asynq client failed")
	}

	return nil
}

// createAsynqScheduler create asynq scheduler
func (s *Server) createAsynqScheduler() error {
	if s.scheduler != nil {
		return nil
	}

	s.scheduler = asynq.NewScheduler(s.redisConnOpt, s.schedulerOpts)
	if s.scheduler == nil {
		LogErrorf("create asynq scheduler failed")
		return errors.New("create asynq scheduler failed")
	}

	return nil
}

// runAsynqScheduler run asynq scheduler
func (s *Server) runAsynqScheduler() error {
	if s.scheduler == nil {
		LogErrorf("asynq scheduler is nil")
		return errors.New("asynq scheduler is nil")
	}

	if err := s.scheduler.Start(); err != nil {
		LogErrorf("asynq scheduler start failed: %s", err.Error())
		return err
	}

	return nil
}

// createAsynqInspector create asynq inspector
func (s *Server) createAsynqInspector() error {
	if s.inspector != nil {
		return nil
	}

	s.inspector = asynq.NewInspector(s.redisConnOpt)
	if s.inspector == nil {
		LogErrorf("create asynq inspector failed")
		return errors.New("create asynq inspector failed")
	}
	return nil
}

func (s *Server) Endpoint() (*url.URL, error) {
	if s.keepaliveServer == nil {
		return nil, errors.New("keepalive server is nil")
	}

	return s.keepaliveServer.Endpoint()
}
