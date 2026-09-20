package thrift

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/url"

	"github.com/apache/thrift/lib/go/thrift"

	kratosTransport "github.com/go-kratos/kratos/v2/transport"

	"github.com/tx7do/kratos-transport/transport"
)

var (
	_ kratosTransport.Server     = (*Server)(nil)
	_ kratosTransport.Endpointer = (*Server)(nil)
)

var (
	ErrInvalidProtocol  = errors.New("invalid protocol")
	ErrInvalidTransport = errors.New("invalid transport")
	ErrInvalidEndpoint  = errors.New("invalid endpoint")
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
	return KindThrift
}

func (s *Server) init(opts ...ServerOption) {
	for _, o := range opts {
		o(s)
	}
}

func (s *Server) Endpoint() (*url.URL, error) {
	if err := s.listenAndEndpoint(); err != nil {
		return nil, err
	}
	return s.endpoint, nil
}

func (s *Server) listenAndEndpoint() error {
	if s.endpoint == nil {
		host, port, err := net.SplitHostPort(s.address)
		if err != nil {
			return err
		}

		if host == "" {
			ip, _ := transport.GetLocalIP()
			host = ip
		}

		addr := host + ":" + fmt.Sprint(port)
		s.endpoint = transport.NewRegistryEndpoint(KindThrift, addr)
	}

	return nil
}

func (s *Server) Start(_ context.Context) error {
	if s.processor == nil {
		return errors.New("thrift processor is nil (use WithProcessor)")
	}

	if s.err = s.listenAndEndpoint(); s.err != nil {
		return s.err
	}

	protocolFactory := createProtocolFactory(s.protocol)
	if protocolFactory == nil {
		return ErrInvalidProtocol
	}

	// TLS：用户通过 WithTLSConfig 提供的配置优先；未提供时才回退默认（向后兼容）
	cfg := &thrift.TConfiguration{}
	if s.tlsConf != nil {
		cfg.TLSConfig = s.tlsConf
	} else {
		cfg.TLSConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	transportFactory := createTransportFactory(cfg, s.buffered, s.framed, s.bufferSize)
	if transportFactory == nil {
		return ErrInvalidTransport
	}

	serverTransport, err := createServerTransport(s.address, s.tlsConf)
	if err != nil {
		return err
	}

	LogInfof("server listening on: %s", s.address)

	s.Server = thrift.NewTSimpleServer4(s.processor, serverTransport, transportFactory, protocolFactory)
	if err := s.Server.Serve(); err != nil {
		return err
	}

	return nil
}

func (s *Server) Stop(_ context.Context) error {
	LogInfo("server stopping...")

	var err error

	if s.Server != nil {
		err = s.Server.Stop()
	}

	LogInfo("server stopped.")

	return err
}
