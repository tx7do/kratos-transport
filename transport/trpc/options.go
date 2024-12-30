package trpc

import (
	trpcServer "trpc.group/trpc-go/trpc-go/server"
)

type ServerOption func(o *Server)

func WithService(serviceName string, service trpcServer.Service) ServerOption {
	return func(s *Server) {
		s.Server.AddService(serviceName, service)
	}
}
