package keepalive

import (
	"crypto/tls"
	"net/url"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type ServerOption func(o *Server)

// WithTLSConfig TLS配置
func WithTLSConfig(tlsConf *tls.Config) ServerOption {
	return func(s *Server) {
		if tlsConf != nil {
			s.grpcOpts = append(s.grpcOpts, grpc.Creds(credentials.NewTLS(tlsConf)))
		}
	}
}

func WithNetwork(network string) ServerOption {
	return func(s *Server) {
		s.network = network
	}
}

func WithAddress(addr string) ServerOption {
	return func(s *Server) {
		s.address = addr
	}
}

func WithEndpoint(endpoint *url.URL) ServerOption {
	return func(s *Server) {
		s.endpoint = endpoint
	}
}

func WithServiceKind(name string) ServerOption {
	return func(s *Server) {
		s.serviceKind = name
	}
}
