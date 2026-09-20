package thrift

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/go-kratos/kratos/v2/registry"
)

type clientOptions struct {
	Client *thrift.TStandardClient

	discovery registry.Discovery
	tlsConf   *tls.Config

	serviceName string
	endpoint    string

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
			LogErrorf("failed to close transport: %v", err)
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

	// TLS 配置：用户通过 WithClientTLSConfig 提供的配置优先；
	// 未提供时才回退到跳过证书校验的默认配置（向后兼容）
	cfg := &thrift.TConfiguration{}
	if cli.tlsConf != nil {
		cfg.TLSConfig = cli.tlsConf
	} else {
		cfg.TLSConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	transportFactory := createTransportFactory(cfg, cli.buffered, cli.framed, cli.bufferSize)
	if transportFactory == nil {
		return nil, ErrInvalidTransport
	}

	endpoint := cli.endpoint
	// 配置了服务发现时，从 selector 解析一个可用节点
	if endpoint == "" && cli.discovery != nil {
		weightNodes, err := cli.discovery.GetService(context.Background(), cli.serviceName)
		if err != nil {
			return nil, fmt.Errorf("discovery get service %q failed: %w", cli.serviceName, err)
		}
		if len(weightNodes) == 0 || len(weightNodes[0].Endpoints) == 0 {
			return nil, ErrInvalidEndpoint
		}
		endpoint = weightNodes[0].Endpoints[0]
	}
	if endpoint == "" {
		return nil, ErrInvalidEndpoint
	}

	clientTransport, err := createClientTransport(transportFactory, endpoint, cli.secure || cli.tlsConf != nil, cfg)
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
