package hptimer

import (
	"context"
	"errors"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	kratosTransport "github.com/go-kratos/kratos/v2/transport"
	"github.com/tx7do/kratos-transport/transport/keepalive"
)

var (
	_ kratosTransport.Server     = (*Server)(nil)
	_ kratosTransport.Endpointer = (*Server)(nil)
)

type Server struct {
	sync.RWMutex

	started  atomic.Bool // 服务是否已启动
	stopping atomic.Bool // 服务是否正在停止

	baseCtx context.Context
	cancel  context.CancelFunc
	err     error

	gracefullyShutdown bool

	keepaliveServer *keepalive.Server
	enableKeepalive bool

	// 高精度定时器引擎
	hpTimer *HighPrecisionTimer

	timerObserver TimerObserver
}

func NewServer(opts ...ServerOption) *Server {
	baseCtx, cancel := context.WithCancel(context.Background())

	srv := &Server{
		baseCtx: baseCtx,
		cancel:  cancel,

		started:  atomic.Bool{},
		stopping: atomic.Bool{},

		gracefullyShutdown: true,
		enableKeepalive:    true,
	}

	srv.init(opts...)

	return srv
}

func (s *Server) init(opts ...ServerOption) {
	for _, o := range opts {
		o(s)
	}
}

// Name returns the name of server
func (s *Server) Name() string {
	return KindHighPrecisionTimer
}

// Start the server
func (s *Server) Start(ctx context.Context) error {
	s.Lock()
	defer s.Unlock()

	if s.started.Load() {
		return errors.New("hptimer server already started")
	}
	if s.err != nil {
		return s.err
	}

	LogInfo("hptimer server starting...")

	// 创建并启动定时器引擎
	s.hpTimer = NewHighPrecisionTimer(s.timerObserver)
	s.hpTimer.Start()

	// 启动 keepalive
	if s.enableKeepalive && s.keepaliveServer != nil {
		if err := s.keepaliveServer.Start(ctx); err != nil {
			s.err = err
			return err
		}
	}

	s.started.Store(true) // 启动成功后设置 started

	LogInfo("hptimer server started successfully")

	return nil
}

// Stop the server
func (s *Server) Stop(ctx context.Context) error {
	s.Lock()
	defer s.Unlock()

	if !s.started.Load() {
		return errors.New("hptimer server not started")
	}
	if s.stopping.Load() {
		return nil
	}

	s.stopping.Store(true)
	defer func() {
		s.stopping.Store(false)
		s.started.Store(false) // 停止后设置 started=false
	}()

	LogInfo("hptimer server stopping...")

	// 1. 取消上下文
	if s.cancel != nil {
		s.cancel()
	}

	// 2. 优雅停止定时器引擎
	stopCtx, stopCancel := context.WithTimeout(ctx, 10*time.Second)
	defer stopCancel()

	wait := make(chan struct{})
	go func() {
		if s.hpTimer != nil {
			s.hpTimer.Stop()
		}
		close(wait)
	}()

	select {
	case <-wait:
		LogInfo("high precision timer stopped gracefully")
	case <-stopCtx.Done():
		LogWarn("hptimer shutdown timeout, force stop")
	}

	// 停止心跳
	if s.keepaliveServer != nil {
		_ = s.keepaliveServer.Stop(ctx)
	}

	LogInfo("hptimer server stopped successfully")
	return nil
}

func (s *Server) Endpoint() (*url.URL, error) {
	if !s.enableKeepalive {
		return &url.URL{}, nil
	}
	if s.keepaliveServer == nil {
		return nil, errors.New("hptimer server keepalive instance is nil")
	}
	return s.keepaliveServer.Endpoint()
}

// AddTask 添加定时任务
func (s *Server) AddTask(task *TimerTask) TimerTaskID {
	if !s.started.Load() {
		return ""
	}
	if s.hpTimer == nil {
		return ""
	}

	return s.hpTimer.AddTask(task)
}

// RemoveTask 删除任务
func (s *Server) RemoveTask(taskID TimerTaskID) bool {
	if !s.started.Load() {
		return false
	}

	if s.hpTimer == nil {
		return false
	}

	return s.hpTimer.RemoveTask(taskID)
}
