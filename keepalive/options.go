package keepalive

import (
	"crypto/tls"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	EnvKeyHost      = "KRATOS_TRANSPORT_KEEPALIVE_HOST"
	EnvKeyInterface = "KRATOS_TRANSPORT_KEEPALIVE_INTERFACE"
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

// SetKeepAliveHost sets the host for the keep-alive service.
func SetKeepAliveHost(host string) {
	_ = os.Setenv(EnvKeyHost, host)
}

// SetKeepAliveInterface sets the network interface for the keep-alive service.
func SetKeepAliveInterface(iface string) {
	_ = os.Setenv(EnvKeyInterface, iface)
}
