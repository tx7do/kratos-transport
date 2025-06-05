package trpc

import (
	trpcServer "trpc.group/trpc-go/trpc-go/server"
)

type ServerOption func(o *Server)

func WithNamespace(namespace string) ServerOption {
	return func(s *Server) {
		s.trpcOptions = append(s.trpcOptions, trpcServer.WithNamespace(namespace))
	}
}

func WithAddress(addr string) ServerOption {
	return func(s *Server) {
		s.trpcOptions = append(s.trpcOptions, trpcServer.WithAddress(addr))
		s.address = addr
	}
}

func WithEnvName(envName string) ServerOption {
	return func(s *Server) {
		s.trpcOptions = append(s.trpcOptions, trpcServer.WithEnvName(envName))
	}
}

func WithContainer(container string) ServerOption {
	return func(s *Server) {
		s.trpcOptions = append(s.trpcOptions, trpcServer.WithContainer(container))
	}
}

func WithSetName(setName string) ServerOption {
	return func(s *Server) {
		s.trpcOptions = append(s.trpcOptions, trpcServer.WithSetName(setName))
	}
}

func WithServiceName(name string) ServerOption {
	return func(s *Server) {
		s.trpcOptions = append(s.trpcOptions, trpcServer.WithServiceName(name))
	}
}
