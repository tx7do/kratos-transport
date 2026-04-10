package cron

import (
	"context"
	"errors"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	kratosTransport "github.com/go-kratos/kratos/v2/transport"
	"github.com/robfig/cron/v3"

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

	cronScheduler *cron.Cron  // cron 调度器
	cronParser    cron.Parser // cron 表达式解析器

	entryIDs sync.Map   // 存储所有任务ID
	cronMu   sync.Mutex // cron 操作锁
}

func NewServer(opts ...ServerOption) *Server {
	baseCtx, cancel := context.WithCancel(context.Background())

	// 初始化 cron 解析器（支持标准表达式 + 秒级）
	cronParser := cron.NewParser(
		cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)

	srv := &Server{
		baseCtx: baseCtx,
		cancel:  cancel,

		started:  atomic.Bool{},
		stopping: atomic.Bool{},

		gracefullyShutdown: true,
		enableKeepalive:    true,

		cronScheduler: cron.New(cron.WithParser(cronParser)),
		entryIDs:      sync.Map{},
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
	return KindCron
}

// Start the server
func (s *Server) Start(ctx context.Context) error {
	s.Lock()
	defer s.Unlock()

	if s.started.Load() {
		return errors.New("cron server already started")
	}
	if s.err != nil {
		return s.err
	}

	LogInfo("cron server starting...")

	// 启动 cron 调度器
	s.cronScheduler.Start()
	s.started.Store(true)

	// 启动 keepalive
	if s.enableKeepalive && s.keepaliveServer != nil {
		if err := s.keepaliveServer.Start(ctx); err != nil {
			s.err = err
			return err
		}
	}

	LogInfo("cron server started successfully")

	return nil
}

// Stop the server
func (s *Server) Stop(ctx context.Context) error {
	s.Lock()
	defer s.Unlock()

	if !s.started.Load() {
		return errors.New("cron server not started")
	}
	if s.stopping.Load() {
		return nil
	}

	s.stopping.Store(true)
	defer func() {
		s.stopping.Store(false)
		s.started.Store(false)
	}()

	LogInfo("cron server stopping...")

	// 取消上下文
	if s.cancel != nil {
		s.cancel()
	}

	// 优雅停止所有定时任务
	ctxStop, cancelStop := context.WithTimeout(ctx, 10*time.Second)
	defer cancelStop()

	stopCh := make(chan struct{})
	go func() {
		// 停止 cron 调度器，等待正在运行的任务执行完成
		s.cronScheduler.Stop()
		close(stopCh)
	}()

	// 等待优雅停止完成
	select {
	case <-stopCh:
		LogInfo("all cron jobs stopped gracefully")
	case <-ctxStop.Done():
		LogWarn("cron server shutdown timeout, force stop")
	}

	// 停止 keepalive
	if s.keepaliveServer != nil {
		_ = s.keepaliveServer.Stop(ctx)
	}

	LogInfo("cron server stopped successfully")
	return nil
}

func (s *Server) Endpoint() (*url.URL, error) {
	if !s.enableKeepalive {
		return &url.URL{}, nil
	}
	if s.keepaliveServer == nil {
		return nil, errors.New("cron server keepalive instance is nil")
	}
	return s.keepaliveServer.Endpoint()
}

// StartTimerJob 添加并启动一个 cron 定时任务
// spec: cron 表达式（支持秒级：秒 分 时 日 月 周）
// cmd: 任务执行函数
func (s *Server) StartTimerJob(spec string, cmd func()) (cron.EntryID, error) {
	if !s.started.Load() {
		return 0, errors.New("cron server not started, please start server first")
	}

	s.cronMu.Lock()
	defer s.cronMu.Unlock()

	// 添加任务
	entryID, err := s.cronScheduler.AddFunc(spec, cmd)
	if err != nil {
		LogErrorf("failed to add cron job: %v", err)
		return 0, err
	}

	// 存储任务ID
	s.entryIDs.Store(entryID, spec)
	LogInfof("cron job started: id=%d, spec=%s", entryID, spec)
	return entryID, nil
}

// StopTimerJob 根据任务ID停止单个任务
func (s *Server) StopTimerJob(entryID cron.EntryID) {
	s.cronMu.Lock()
	defer s.cronMu.Unlock()

	// 移除任务
	s.cronScheduler.Remove(entryID)
	// 删除记录
	if spec, ok := s.entryIDs.LoadAndDelete(entryID); ok {
		LogInfof("cron job stopped: id=%d, spec=%s", entryID, spec)
	}
}

// StopAllJobs 停止所有定时任务
func (s *Server) StopAllJobs() {
	s.cronMu.Lock()
	defer s.cronMu.Unlock()

	count := 0
	s.entryIDs.Range(func(key, _ interface{}) bool {
		if entryID, ok := key.(cron.EntryID); ok {
			s.cronScheduler.Remove(entryID)
			count++
		}
		return true
	})

	s.entryIDs = sync.Map{}
	LogInfof("all cron jobs stopped, total count: %d", count)
}

// GetJobCount 获取当前运行的任务数量
func (s *Server) GetJobCount() int {
	count := 0
	s.entryIDs.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}
