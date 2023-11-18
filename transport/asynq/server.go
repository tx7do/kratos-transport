package asynq

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"

	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/go-kratos/kratos/v2/transport"

	"github.com/hibiken/asynq"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/utils"
)

var (
	_ transport.Server     = (*Server)(nil)
	_ transport.Endpointer = (*Server)(nil)
)

type Server struct {
	sync.RWMutex
	started bool

	baseCtx context.Context
	err     error

	asynqServer    *asynq.Server
	asynqClient    *asynq.Client
	asynqScheduler *asynq.Scheduler

	mux           *asynq.ServeMux
	asynqConfig   asynq.Config
	redisOpt      asynq.RedisClientOpt
	schedulerOpts *asynq.SchedulerOpts

	keepAlive       *utils.KeepAliveService
	enableKeepAlive bool

	codec encoding.Codec

	entryIDs    map[string]string
	mtxEntryIDs sync.RWMutex
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		baseCtx: context.Background(),
		started: false,

		redisOpt: asynq.RedisClientOpt{
			Addr: defaultRedisAddress,
			DB:   0,
		},
		asynqConfig: asynq.Config{
			Concurrency: 20,
			Logger:      newLogger(),
		},
		schedulerOpts: &asynq.SchedulerOpts{},
		mux:           asynq.NewServeMux(),

		keepAlive:       utils.NewKeepAliveService(nil),
		enableKeepAlive: true,

		codec: encoding.GetCodec("json"),

		entryIDs:    make(map[string]string),
		mtxEntryIDs: sync.RWMutex{},
	}

	srv.init(opts...)

	return srv
}

func (s *Server) Name() string {
	return "asynq"
}

func (s *Server) Endpoint() (*url.URL, error) {
	if s.err != nil {
		return nil, s.err
	}

	return s.keepAlive.Endpoint()
}

// RegisterSubscriber register task subscriber
func (s *Server) RegisterSubscriber(taskType string, handler MessageHandler, binder Binder) error {
	return s.handleFunc(taskType, func(ctx context.Context, task *asynq.Task) error {
		var payload MessagePayload

		if binder != nil {
			payload = binder()
		} else {
			payload = task.Payload()
		}

		if err := broker.Unmarshal(s.codec, task.Payload(), &payload); err != nil {
			LogErrorf("unmarshal message failed: %s", err)
			return err
		}

		if err := handler(task.Type(), payload); err != nil {
			LogErrorf("handle message failed: %s", err)
			return err
		}

		return nil
	})
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

func (s *Server) handleFunc(pattern string, handler func(context.Context, *asynq.Task) error) error {
	if s.started {
		LogErrorf("handleFunc [%s] failed", pattern)
		return errors.New("cannot handle func, server already started")
	}
	s.mux.HandleFunc(pattern, handler)
	return nil
}

// NewTask enqueue a new task
func (s *Server) NewTask(typeName string, msg broker.Any, opts ...asynq.Option) error {
	if s.asynqClient == nil {
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

	taskInfo, err := s.asynqClient.Enqueue(task, opts...)
	if err != nil {
		LogErrorf("[%s] Enqueue failed: %s", typeName, err.Error())
		return err
	}

	LogDebugf("[%s] enqueued task: id=%s queue=%s", typeName, taskInfo.ID, taskInfo.Queue)

	return nil
}

// NewPeriodicTask enqueue a new crontab task
func (s *Server) NewPeriodicTask(cronSpec, typeName string, msg broker.Any, opts ...asynq.Option) (string, error) {
	if s.asynqScheduler == nil {
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

	task := asynq.NewTask(typeName, payload, opts...)
	if task == nil {
		return "", errors.New("new task failed")
	}

	entryID, err := s.asynqScheduler.Register(cronSpec, task, opts...)
	if err != nil {
		LogErrorf("[%s] enqueue periodic task failed: %s", typeName, err.Error())
		return "", err
	}

	s.addPeriodicTaskEntryID(typeName, entryID)

	LogDebugf("[%s]  registered an entry: id=%q", typeName, entryID)

	return entryID, nil
}

// RemovePeriodicTask remove periodic task
func (s *Server) RemovePeriodicTask(typeName string) error {
	entryID := s.QueryPeriodicTaskEntryID(typeName)
	if entryID == "" {
		return errors.New(fmt.Sprintf("[%s] periodic task not exist", typeName))
	}

	if err := s.unregisterPeriodicTask(entryID); err != nil {
		LogErrorf("[%s] dequeue periodic task failed: %s", entryID, err.Error())
		return err
	}

	s.removePeriodicTaskEntryID(typeName)

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

func (s *Server) unregisterPeriodicTask(entryID string) error {
	if s.asynqScheduler == nil {
		return nil
	}

	if err := s.asynqScheduler.Unregister(entryID); err != nil {
		LogErrorf("[%s] dequeue periodic task failed: %s", entryID, err.Error())
		return err
	}

	return nil
}

func (s *Server) addPeriodicTaskEntryID(typeName, entryID string) {
	s.mtxEntryIDs.Lock()
	defer s.mtxEntryIDs.Unlock()

	s.entryIDs[typeName] = entryID
}

func (s *Server) removePeriodicTaskEntryID(typeName string) {
	s.mtxEntryIDs.Lock()
	defer s.mtxEntryIDs.Unlock()

	delete(s.entryIDs, typeName)
}

func (s *Server) QueryPeriodicTaskEntryID(typeName string) string {
	s.mtxEntryIDs.RLock()
	defer s.mtxEntryIDs.RUnlock()

	entryID, ok := s.entryIDs[typeName]
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

	if s.started {
		return nil
	}

	if err := s.runAsynqScheduler(); err != nil {
		LogError("run async scheduler failed", err)
		return err
	}

	if err := s.runAsynqServer(); err != nil {
		LogError("run async server failed", err)
		return err
	}

	if s.enableKeepAlive {
		go func() {
			_ = s.keepAlive.Start()
		}()
	}

	LogInfof("server listening on: %s", s.redisOpt.Addr)

	s.baseCtx = ctx
	s.started = true

	return nil
}

// Stop the server
func (s *Server) Stop(_ context.Context) error {
	LogInfo("server stopping")
	s.started = false

	if s.asynqClient != nil {
		_ = s.asynqClient.Close()
		s.asynqClient = nil
	}

	if s.asynqServer != nil {
		s.asynqServer.Shutdown()
		s.asynqServer = nil
	}

	if s.asynqScheduler != nil {
		s.asynqScheduler.Shutdown()
		s.asynqScheduler = nil
	}

	return nil
}

func (s *Server) init(opts ...ServerOption) {
	for _, o := range opts {
		o(s)
	}
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
}

// createAsynqServer create asynq server
func (s *Server) createAsynqServer() error {
	if s.asynqServer != nil {
		return nil
	}

	s.asynqServer = asynq.NewServer(s.redisOpt, s.asynqConfig)
	if s.asynqServer == nil {
		LogErrorf("create asynq server failed")
		return errors.New("create asynq server failed")
	}
	return nil
}

// runAsynqServer run asynq server
func (s *Server) runAsynqServer() error {
	if s.asynqServer == nil {
		LogErrorf("asynq server is nil")
		return errors.New("asynq server is nil")
	}

	if err := s.asynqServer.Run(s.mux); err != nil {
		LogErrorf("asynq server run failed: %s", err.Error())
		return err
	}
	return nil
}

// createAsynqClient create asynq client
func (s *Server) createAsynqClient() error {
	if s.asynqClient != nil {
		return nil
	}

	s.asynqClient = asynq.NewClient(s.redisOpt)
	if s.asynqClient == nil {
		LogErrorf("create asynq client failed")
		return errors.New("create asynq client failed")
	}

	return nil
}

// createAsynqScheduler create asynq scheduler
func (s *Server) createAsynqScheduler() error {
	if s.asynqScheduler != nil {
		return nil
	}

	s.asynqScheduler = asynq.NewScheduler(s.redisOpt, s.schedulerOpts)
	if s.asynqScheduler == nil {
		LogErrorf("create asynq scheduler failed")
		return errors.New("create asynq scheduler failed")
	}

	return nil
}

// runAsynqScheduler run asynq scheduler
func (s *Server) runAsynqScheduler() error {
	if s.asynqScheduler == nil {
		LogErrorf("asynq scheduler is nil")
		return errors.New("asynq scheduler is nil")
	}

	if err := s.asynqScheduler.Start(); err != nil {
		LogErrorf("asynq scheduler start failed: %s", err.Error())
		return err
	}

	return nil
}
