package thrift

import (
	"crypto/tls"
	"github.com/apache/thrift/lib/go/thrift"
	"github.com/go-kratos/kratos/v2/registry"
)

type clientOptions struct {
	Client *thrift.TStandardClient

	discovery registry.Discovery
	tlsConf   *tls.Config

	address string

	protocol string

	buffered   bool
	framed     bool
	bufferSize int

	secure bool
}

func Dial(opts ...ClientOption) (*thrift.TStandardClient, error) {
	return dial(opts...)
}

func dial(opts ...ClientOption) (*thrift.TStandardClient, error) {
	cli := &clientOptions{
		bufferSize: 8192,
		buffered:   false,
		framed:     false,
		protocol:   "binary",
		secure:     false,
	}

	for _, o := range opts {
		o(cli)
	}

	protocolFactory := createProtocolFactory(cli.protocol)
	if protocolFactory == nil {
		return nil, ErrInvalidProtocol
	}

	cfg := &thrift.TConfiguration{
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	transportFactory := createTransportFactory(cfg, cli.buffered, cli.framed, cli.bufferSize)
	if transportFactory == nil {
		return nil, ErrInvalidTransport
	}

	clientTransport, err := createClientTransport(transportFactory, cli.address, cli.secure, cfg)
	if err != nil {
		return nil, err
	}

	iProto := protocolFactory.GetProtocol(clientTransport)
	oProto := protocolFactory.GetProtocol(clientTransport)

	return thrift.NewTStandardClient(iProto, oProto), nil
}
