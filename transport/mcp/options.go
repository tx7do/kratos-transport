package mcp

import "github.com/mark3labs/mcp-go/server"

type ServerOption func(o *Server)

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
