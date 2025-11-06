package mcp

import (
	"context"
	"errors"
	"net/url"
	"sync"
	"sync/atomic"

	"github.com/go-kratos/kratos/v2/log"
	kratosTransport "github.com/go-kratos/kratos/v2/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/tx7do/kratos-transport/transport/keepalive"

	"github.com/mark3labs/mcp-go/server"
)

var (
	_ kratosTransport.Server     = (*Server)(nil)
	_ kratosTransport.Endpointer = (*Server)(nil)
)

type Server struct {
	mu      sync.RWMutex
	started atomic.Bool

	baseCtx context.Context
	err     error

	serverName    string
	serverVersion string

	keepaliveServer *keepalive.Server
	mcpServer       *server.MCPServer

	mcpOpts []server.ServerOption
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		baseCtx: context.Background(),
		started: atomic.Bool{},
	}

	srv.init(opts...)

	return srv
}

func (s *Server) init(opts ...ServerOption) {
	for _, o := range opts {
		o(s)
	}

	// Create a new Keep Alive Server
	s.keepaliveServer = keepalive.NewServer(
		keepalive.WithServiceKind(KindMCP),
	)

	// Create a new MCP server
	s.mcpServer = server.NewMCPServer(s.serverName, s.serverVersion, s.mcpOpts...)
}

func (s *Server) Name() string {
	return KindMCP
}

func (s *Server) Start(ctx context.Context) error {
	s.mu.RLock()
	if s.err != nil {
		s.mu.RUnlock()
		return s.err
	}
	s.mu.RUnlock()

	if s.started.Load() {
		LogWarn("MCP server already started")
		return nil
	}

	// Start the keep alive server
	if s.keepaliveServer != nil {
		go func() {
			if err := s.keepaliveServer.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
				s.mu.Lock()
				s.err = errors.Join(s.err, errors.New("keepalive server start failed: "+err.Error()))
				s.mu.Unlock()
				LogErrorf("keepalive server start failed, err: %v", err)
			}
		}()
	}

	// Start the MCP server
	if err := s.startMCPServer(); err != nil {
		s.mu.Lock()
		s.err = errors.Join(s.err, err)
		s.mu.Unlock()
		_ = s.Stop(ctx)
		return err
	}

	s.baseCtx = ctx
	s.started.Store(true)

	LogInfof("MCP server started, [%s][%s]", s.serverName, s.serverVersion)

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	if !s.started.Load() {
		log.Warn("MCP server already stopped")
		return nil
	}

	LogInfof("MCP server stopping, name: %s", s.serverName)

	s.started.Store(false)

	s.err = nil

	if s.keepaliveServer != nil {
		if s.err = s.keepaliveServer.Stop(ctx); s.err != nil {
			LogError("keepalive server stop failed", s.err)
		}
		s.keepaliveServer = nil
	}

	LogInfo("server stopped.")

	return s.err
}

func (s *Server) Endpoint() (*url.URL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.keepaliveServer == nil {
		return nil, errors.New("keepalive server is nil")
	}

	return s.keepaliveServer.Endpoint()
}

func (s *Server) RegisterHandler(tool mcp.Tool, handler server.ToolHandlerFunc) error {
	if s.mcpServer == nil {
		return errors.New("mcp server is nil")
	}

	s.mcpServer.AddTool(tool, handler)

	return nil
}

func (s *Server) startMCPServer() error {
	if s.mcpServer == nil {
		return errors.New("MCP server instance is nil")
	}

	if err := server.ServeStdio(s.mcpServer); err != nil {
		log.Errorw("MCP server start failed", "err", err)
		return errors.New("start MCP server: " + err.Error())
	}

	return nil
}

func (s *Server) waitGroup(wg *sync.WaitGroup, ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
