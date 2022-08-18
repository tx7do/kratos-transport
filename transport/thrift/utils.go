package thrift

import (
	"crypto/tls"
	"github.com/apache/thrift/lib/go/thrift"
)

func createProtocolFactory(protocol string) thrift.TProtocolFactory {
	switch protocol {
	case "compact":
		return thrift.NewTCompactProtocolFactoryConf(nil)
	case "simplejson":
		return thrift.NewTSimpleJSONProtocolFactoryConf(nil)
	case "json":
		return thrift.NewTJSONProtocolFactory()
	case "binary", "":
		return thrift.NewTBinaryProtocolFactoryConf(nil)
	default:
		return nil
	}
}

func createTransportFactory(cfg *thrift.TConfiguration, buffered, framed bool, bufferSize int) thrift.TTransportFactory {
	var transportFactory thrift.TTransportFactory

	if buffered {
		transportFactory = thrift.NewTBufferedTransportFactory(bufferSize)
	} else {
		transportFactory = thrift.NewTTransportFactory()
	}

	if framed {
		transportFactory = thrift.NewTFramedTransportFactoryConf(transportFactory, cfg)
	}

	return transportFactory
}

func createServerTransport(address string, tlsConf *tls.Config) (thrift.TServerTransport, error) {
	if tlsConf != nil {
		return thrift.NewTSSLServerSocket(address, tlsConf)
	} else {
		return thrift.NewTServerSocket(address)
	}
}

func createClientTransport(transportFactory thrift.TTransportFactory, address string, secure bool, cfg *thrift.TConfiguration) (thrift.TTransport, error) {
	var transport thrift.TTransport
	if secure {
		transport = thrift.NewTSSLSocketConf(address, cfg)
	} else {
		transport = thrift.NewTSocketConf(address, cfg)
	}
	transport, err := transportFactory.GetTransport(transport)
	if err != nil {
		return nil, err
	}
	if err := transport.Open(); err != nil {
		return nil, err
	}
	return transport, nil
}
