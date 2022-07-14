package rocketmq

import (
	"github.com/tx7do/kratos-transport/broker"
)

type nameServersKey struct{}
type nameServerUrlKey struct{}
type accessKey struct{}
type secretKey struct{}
type retryCountKey struct{}
type namespaceKey struct{}

func WithNameServer(addrs []string) broker.Option {
	return broker.OptionContextWithValue(nameServersKey{}, addrs)
}
func WithNameServerDomain(uri string) broker.Option {
	return broker.OptionContextWithValue(nameServerUrlKey{}, uri)
}

func WithAccessKey(key string) broker.Option {
	return broker.OptionContextWithValue(accessKey{}, key)
}
func WithSecretKey(key string) broker.Option {
	return broker.OptionContextWithValue(secretKey{}, key)
}
func WithRetryCount(count int) broker.Option {
	return broker.OptionContextWithValue(retryCountKey{}, count)
}
func WithNamespace(name string) broker.Option {
	return broker.OptionContextWithValue(namespaceKey{}, name)
}
