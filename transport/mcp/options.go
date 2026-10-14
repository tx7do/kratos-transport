package mcp

import "github.com/mark3labs/mcp-go/server"

type ServerType string

type ServerOption func(o *Server)

const (
	ServerTypeSSE       ServerType = "SSE"
	ServerTypeHTTP      ServerType = "HTTP"
	ServerTypeStdio     ServerType = "STDIO"
	ServerTypeInProcess ServerType = "IN_PROCESS"
)

func WithServerName(name string) ServerOption {
	return func(s *Server) {
		s.serverName = name
	}
}

func WithServerVersion(version string) ServerOption {
	return func(s *Server) {
		s.serverVersion = version
	}
}

func WithMCPServerOptions(opts ...server.ServerOption) ServerOption {
	return func(s *Server) {
		s.mcpOpts = append(s.mcpOpts, opts...)
	}
}

func WithMCPServeType(serverType ServerType) ServerOption {
	return func(s *Server) {
		s.serverType = serverType
	}
}

func WithMCPServeAddress(addr string) ServerOption {
	return func(s *Server) {
		s.serverAddr = addr
	}
}

// WithMCPHTTPServerOptions 透传 StreamableHTTP 服务选项（如 WithHTTPContextFunc
// 鉴权注入、WithRecovery 等）。仅 ServerTypeHTTP 生效。
func WithMCPHTTPServerOptions(opts ...server.StreamableHTTPOption) ServerOption {
	return func(s *Server) {
		s.httpOpts = append(s.httpOpts, opts...)
	}
}

// WithMCPSSEServerOptions 透传 SSE 服务选项。仅 ServerTypeSSE 生效。
func WithMCPSSEServerOptions(opts ...server.SSEOption) ServerOption {
	return func(s *Server) {
		s.sseOpts = append(s.sseOpts, opts...)
	}
}
