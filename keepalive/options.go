package keepalive

import (
	"crypto/tls"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type ServiceOption func(o *Service)

// WithTLSConfig TLS配置
func WithTLSConfig(tlsConf *tls.Config) ServiceOption {
	return func(s *Service) {
		if tlsConf != nil {
			s.grpcOpts = append(s.grpcOpts, grpc.Creds(credentials.NewTLS(tlsConf)))
		}
	}
}

func WithHost(host string) ServiceOption {
	return func(s *Service) {
		s.host = host
	}
}
