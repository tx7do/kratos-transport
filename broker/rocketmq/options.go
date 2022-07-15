package rocketmq

import (
	"github.com/tx7do/kratos-transport/broker"
)

///
/// Option
///

type nameServersKey struct{}
type nameServerUrlKey struct{}
type accessKey struct{}
type secretKey struct{}
type retryCountKey struct{}
type namespaceKey struct{}
type instanceKey struct{}

func WithNameServer(addrs []string) broker.Option {
	return broker.OptionContextWithValue(nameServersKey{}, addrs)
}
func WithNameServerDomain(uri string) broker.Option {
	return broker.OptionContextWithValue(nameServerUrlKey{}, uri)
}

func WithAccessKey(key string) broker.Option {
	return broker.OptionContextWithValue(accessKey{}, key)
}

func WithSecretKey(secret string) broker.Option {
	return broker.OptionContextWithValue(secretKey{}, secret)
}

func WithRetryCount(count int) broker.Option {
	return broker.OptionContextWithValue(retryCountKey{}, count)
}

func WithNamespace(name string) broker.Option {
	return broker.OptionContextWithValue(namespaceKey{}, name)
}

func WithInstanceName(name string) broker.Option {
	return broker.OptionContextWithValue(instanceKey{}, name)
}

///
/// PublishOption
///

type compressKey struct{}
type batchKey struct{}

func WithCompressPublish(compress bool) broker.PublishOption {
	return broker.PublishContextWithValue(compressKey{}, compress)
}

func WithBatchPublish(batch bool) broker.PublishOption {
	return broker.PublishContextWithValue(batchKey{}, batch)
}
