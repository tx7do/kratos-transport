package thrift

import (
	"crypto/tls"
	"github.com/apache/thrift/lib/go/thrift"
	"github.com/go-kratos/kratos/v2/registry"
)

type ServerOption func(o *Server)

func WithAddress(addr string) ServerOption {
	return func(s *Server) {
		s.address = addr
	}
}

func WithTLSConfig(c *tls.Config) ServerOption {
	return func(s *Server) {
		s.tlsConf = c
	}
}

func WithProcessor(processor thrift.TProcessor) ServerOption {
	return func(s *Server) {
		s.processor = processor
	}
}

func WithProtocol(protocol string) ServerOption {
	return func(s *Server) {
		s.protocol = protocol
	}
}

func WithTransportConfig(buffered, framed bool, bufferSize int) ServerOption {
	return func(s *Server) {
		s.buffered = buffered
		s.framed = framed
		s.bufferSize = bufferSize
	}
}

////////////////////////////////////////////////////////////////////////////////

type ClientOption func(o *clientOptions)

// WithDiscovery with client discovery.
func WithDiscovery(d registry.Discovery) ClientOption {
	return func(o *clientOptions) {
		o.discovery = d
	}
}

// WithEndpoint with client endpoint.
func WithEndpoint(endpoint string) ClientOption {
	return func(o *clientOptions) {
		o.endpoint = endpoint
	}
}

// WithClientTLSConfig with tls config.
func WithClientTLSConfig(c *tls.Config) ClientOption {
	return func(o *clientOptions) {
		o.tlsConf = c
	}
}

func WithClientProtocol(protocol string) ClientOption {
	return func(o *clientOptions) {
		o.protocol = protocol
	}
}

func WithClientTransportConfig(buffered, framed bool, bufferSize int) ClientOption {
	return func(o *clientOptions) {
		o.buffered = buffered
		o.framed = framed
		o.bufferSize = bufferSize
	}
}
