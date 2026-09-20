package mcp

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/go-kratos/kratos/v2/log"
	kratosTransport "github.com/go-kratos/kratos/v2/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/tx7do/kratos-transport/transport/keepalive"

	"github.com/mark3labs/mcp-go/server"
)

const (
	DefaultMCPServerName    = "MCP Server"
	DefaultMCPServerVersion = "1.0.0"
	DefaultMCPServerAddress = ":8080"
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
	sseServer       *server.SSEServer
	httpServer      *server.StreamableHTTPServer
	endpoint        *url.URL

	mcpOpts []server.ServerOption

	serverType ServerType
	serverAddr string
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		baseCtx:       context.Background(),
		started:       atomic.Bool{},
		serverType:    ServerTypeStdio,
		serverVersion: DefaultMCPServerVersion,
		serverName:    DefaultMCPServerName,
		serverAddr:    DefaultMCPServerAddress,
	}

	srv.init(opts...)

	return srv
}

func (s *Server) init(opts ...ServerOption) {
	for _, o := range opts {
		o(s)
	}

	switch s.serverType {
	case ServerTypeSSE, ServerTypeHTTP:
		log.Infof("MCP server type set to %s, address: %s", s.serverType, s.serverAddr)
	case ServerTypeInProcess:
		log.Info("MCP server type set to IN_PROCESS")
		s.newKeepaliveServer()
	case ServerTypeStdio:
		log.Info("MCP server type set to STDIO")
		fallthrough
	default:
		log.Warnf("Unsupported MCP server type: %s, defaulting to STDIO", s.serverType)
		s.serverType = ServerTypeStdio
		s.newKeepaliveServer()
	}

	if s.keepaliveServer != nil {
		s.endpoint, _ = s.keepaliveServer.Endpoint()
	} else {
		host := s.serverAddr
		if host == "" {
			host = DefaultMCPServerAddress
		}
		if strings.HasPrefix(host, ":") {
			host = "localhost" + host
		}
		s.endpoint = &url.URL{
			Scheme: "http",
			Host:   host,
		}
	}

	// Create a new MCP server
	s.mcpServer = server.NewMCPServer(s.serverName, s.serverVersion, s.mcpOpts...)
}

func (s *Server) Name() string {
	return KindMCP
}

func (s *Server) Start(ctx context.Context) error {
	s.mu.RLock()
	if s.err != nil {
		e := s.err
		s.mu.RUnlock()
		return e
	}
	s.mu.RUnlock()

	if s.started.Load() {
		LogWarn("MCP server already started")
		return nil
	}

	// Start the keep alive server
	s.startKeepaliveServer(ctx)

	// Start the MCP server
	go func() {
		if err := s.startMCPServer(); err != nil {
			s.setErr(err)
			s.stopKeepaliveServer(ctx)
		}
	}()

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

	s.stopKeepaliveServer(ctx)

	// 停掉 SSE/HTTP 服务本体：否则 Stop 后端口仍被占用，
	// 再次 Start 会在同地址起第二个实例
	s.mu.Lock()
	sseServer := s.sseServer
	httpServer := s.httpServer
	s.sseServer = nil
	s.httpServer = nil
	s.setErrLocked(nil) // 清 sticky err，避免瞬时错误导致后续 Start 永久失败
	s.mu.Unlock()

	if sseServer != nil {
		if err := sseServer.Shutdown(ctx); err != nil {
			LogErrorf("sse server shutdown failed: %s", err.Error())
		}
	}
	if httpServer != nil {
		if err := httpServer.Shutdown(ctx); err != nil {
			LogErrorf("http server shutdown failed: %s", err.Error())
		}
	}

	s.mu.RLock()
	err := s.err
	s.mu.RUnlock()

	if err != nil {
		LogError("server stopped with error", err)
	} else {
		LogInfo("server stopped.")
	}

	return err
}

func (s *Server) Endpoint() (*url.URL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.endpoint == nil {
		return nil, errors.New("endpoint is nil")
	}

	return s.endpoint, nil
}

func (s *Server) RegisterHandler(tool mcp.Tool, handler server.ToolHandlerFunc) error {
	if s.mcpServer == nil {
		return errors.New("mcp server is nil")
	}

	s.mcpServer.AddTool(tool, handler)

	return nil
}

func (s *Server) RegisterHandlerWithJsonString(jsonString string, handler server.ToolHandlerFunc) error {
	if s.mcpServer == nil {
		return errors.New("mcp server is nil")
	}

	tool, err := LoadToolFromJsonString(jsonString)
	if err != nil {
		return err
	}

	return s.RegisterHandler(tool, handler)
}

func (s *Server) RegisterHandlerWithJsonSchema(name, description string, jsonSchemaString string, handler server.ToolHandlerFunc) error {
	if s.mcpServer == nil {
		return errors.New("mcp server is nil")
	}

	raw := toRawMessage(jsonSchemaString)

	tool := mcp.NewToolWithRawSchema(name, description, raw)

	return s.RegisterHandler(tool, handler)
}

func (s *Server) startMCPServer() error {
	if s.mcpServer == nil {
		return errors.New("MCP server instance is nil")
	}

	switch s.serverType {
	case ServerTypeStdio:
		if err := server.ServeStdio(s.mcpServer); err != nil {
			LogErrorf("MCP server start failed: %s", err.Error())
			return errors.New("start MCP server: " + err.Error())
		}

	case ServerTypeSSE:
		sseServer := server.NewSSEServer(s.mcpServer)
		// 保存句柄供 Stop 关闭（此前是局部变量，Stop 后端口仍在服务）
		s.sseServer = sseServer
		if err := sseServer.Start(s.serverAddr); err != nil {
			s.sseServer = nil
			// 不能用 Fatalf：会 os.Exit 杀死整个 kratos 应用
			LogErrorf("MCP server start failed: %s", err.Error())
			return errors.New("start MCP server: " + err.Error())
		}

	case ServerTypeHTTP:
		httpServer := server.NewStreamableHTTPServer(s.mcpServer)
		s.httpServer = httpServer
		if err := httpServer.Start(s.serverAddr); err != nil {
			s.httpServer = nil
			LogErrorf("MCP server start failed: %s", err.Error())
			return errors.New("start MCP server: " + err.Error())
		}

	case ServerTypeInProcess:

	default:
		return errors.New("unsupported MCP server type: " + string(s.serverType))
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

// Create a new Keep Alive Server
func (s *Server) newKeepaliveServer() {
	s.keepaliveServer = keepalive.NewServer(
		keepalive.WithServiceKind(KindMCP),
	)
}

func (s *Server) startKeepaliveServer(ctx context.Context) {
	// Stop 置 nil 后重建（keepalive 的 stopReq 闩锁不可复位）
	if s.keepaliveServer == nil {
		s.keepaliveServer = keepalive.NewServer(keepalive.WithServiceKind(KindMCP))
	}

	// 捕获局部引用：并发 Stop 置 nil 后 goroutine 内不会 nil panic
	if ka := s.keepaliveServer; ka != nil {
		go func() {
			if err := ka.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
				s.mu.Lock()
				s.err = errors.Join(s.err, errors.New("keepalive server start failed: "+err.Error()))
				s.mu.Unlock()
				LogErrorf("keepalive server start failed, err: %v", err)
			}
		}()
	}
}

func (s *Server) stopKeepaliveServer(ctx context.Context) {
	if s.keepaliveServer != nil {
		s.mu.Lock()
		s.err = s.keepaliveServer.Stop(ctx)
		s.mu.Unlock()
		if s.err != nil {
			LogError("keepalive server stop failed", s.err)
		}
		s.keepaliveServer = nil
	}
}

func (s *Server) setErr(err error) {
	if err == nil {
		return
	}
	s.mu.Lock()
	s.err = errors.Join(s.err, err)
	s.mu.Unlock()
}

// setErrLocked 在已持锁情况下设置/清空错误（nil 清空 sticky err，避免 Start 永久失败）
func (s *Server) setErrLocked(err error) {
	s.err = err
}
