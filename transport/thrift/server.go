package thrift

import (
	"context"
	"crypto/tls"
	"errors"
	"net/url"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
)

var (
	_ transport.Server     = (*Server)(nil)
	_ transport.Endpointer = (*Server)(nil)
)

var (
	ErrInvalidProtocol  = errors.New("invalid protocol")
	ErrInvalidTransport = errors.New("invalid transport")
)

type Server struct {
	Server *thrift.TSimpleServer

	tlsConf  *tls.Config
	endpoint *url.URL

	address string

	protocol string

	buffered   bool
	framed     bool
	bufferSize int

	processor thrift.TProcessor

	err error
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		bufferSize: 8192,
		buffered:   false,
		framed:     false,
		protocol:   "binary",
	}

	srv.init(opts...)

	return srv
}

func (s *Server) Name() string {
	return "thrift"
}

func (s *Server) init(opts ...ServerOption) {
	for _, o := range opts {
		o(s)
	}

	s.endpoint, s.err = url.Parse("tcp://" + s.address)
}

func (s *Server) Endpoint() (*url.URL, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.endpoint, nil
}

func (s *Server) Start(ctx context.Context) error {
	if s.err != nil {
		return s.err
	}

	protocolFactory := createProtocolFactory(s.protocol)
	if protocolFactory == nil {
		return ErrInvalidProtocol
	}

	cfg := &thrift.TConfiguration{
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	transportFactory := createTransportFactory(cfg, s.buffered, s.framed, s.bufferSize)
	if transportFactory == nil {
		return ErrInvalidTransport
	}

	serverTransport, err := createServerTransport(s.address, s.tlsConf)
	if err != nil {
		return err
	}

	log.Infof("[redis] server listening on: %s", s.address)

	s.Server = thrift.NewTSimpleServer4(s.processor, serverTransport, transportFactory, protocolFactory)
	if err := s.Server.Serve(); err != nil {
		return err
	}

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	log.Info("[thrift] server stopping")

	if s.Server != nil {
		return s.Server.Stop()
	}

	return nil
}
