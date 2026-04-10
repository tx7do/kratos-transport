package hptimer

type ServerOption func(o *Server)

func WithEnableKeepAlive(enable bool) ServerOption {
	return func(s *Server) {
		s.enableKeepalive = enable
	}
}

// WithGracefullyShutdown 设置是否优雅关闭
func WithGracefullyShutdown(enable bool) ServerOption {
	return func(s *Server) {
		s.gracefullyShutdown = enable
	}
}

func WithTimerObserver(observer TimerObserver) ServerOption {
	return func(s *Server) {
		s.timerObserver = observer
	}
}
