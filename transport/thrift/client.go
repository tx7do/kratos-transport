package thrift

import (
	"crypto/tls"
	"github.com/apache/thrift/lib/go/thrift"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
)

type clientOptions struct {
	Client *thrift.TStandardClient

	discovery registry.Discovery
	tlsConf   *tls.Config

	endpoint string

	protocol string

	buffered   bool
	framed     bool
	bufferSize int

	secure bool
}

type Connection struct {
	Client    *thrift.TStandardClient
	Transport thrift.TTransport
}

func (c *Connection) Close() {
	if c.Transport != nil {
		err := c.Transport.Close()
		if err != nil {
			log.Error("failed to close transport: %v", err)
		}
	}
}

func Dial(opts ...ClientOption) (*Connection, error) {
	return dial(opts...)
}

func dial(opts ...ClientOption) (*Connection, error) {
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

	clientTransport, err := createClientTransport(transportFactory, cli.endpoint, cli.secure, cfg)
	if err != nil {
		return nil, err
	}

	iProto := protocolFactory.GetProtocol(clientTransport)
	oProto := protocolFactory.GetProtocol(clientTransport)

	return &Connection{
		Client:    thrift.NewTStandardClient(iProto, oProto),
		Transport: clientTransport,
	}, nil
}
