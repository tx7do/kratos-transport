package rocketmq

import (
	"github.com/tx7do/kratos-transport/broker"
)

///
/// Option
///

type enableAliyunHttpKey struct{}
type enableTraceKey struct{}
type nameServersKey struct{}
type nameServerUrlKey struct{}
type accessKey struct{}
type secretKey struct{}
type securityTokenKey struct{}
type retryCountKey struct{}
type namespaceKey struct{}
type instanceNameKey struct{}
type groupNameKey struct{}

func WithAliyunHttpSupport() broker.Option {
	return broker.OptionContextWithValue(enableAliyunHttpKey{}, true)
}

func WithEnableTrace() broker.Option {
	return broker.OptionContextWithValue(enableTraceKey{}, true)
}

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
func WithSecurityToken(token string) broker.Option {
	return broker.OptionContextWithValue(securityTokenKey{}, token)
}

func WithRetryCount(count int) broker.Option {
	return broker.OptionContextWithValue(retryCountKey{}, count)
}

func WithNamespace(ns string) broker.Option {
	return broker.OptionContextWithValue(namespaceKey{}, ns)
}

func WithInstanceName(name string) broker.Option {
	return broker.OptionContextWithValue(instanceNameKey{}, name)
}

func WithGroupName(name string) broker.Option {
	return broker.OptionContextWithValue(groupNameKey{}, name)
}

///
/// PublishOption
///

type compressKey struct{}
type batchKey struct{}
type propertiesKey struct{}
type delayTimeLevelKey struct{}
type tagsKey struct{}
type keysKey struct{}
type shardingKeyKey struct{}

func WithCompress(compress bool) broker.PublishOption {
	return broker.PublishContextWithValue(compressKey{}, compress)
}

func WithBatch(batch bool) broker.PublishOption {
	return broker.PublishContextWithValue(batchKey{}, batch)
}

func WithProperties(properties map[string]string) broker.PublishOption {
	return broker.PublishContextWithValue(propertiesKey{}, properties)
}

func WithDelayTimeLevel(level int) broker.PublishOption {
	return broker.PublishContextWithValue(delayTimeLevelKey{}, level)
}

func WithTag(tags string) broker.PublishOption {
	return broker.PublishContextWithValue(tagsKey{}, tags)
}

func WithKeys(keys []string) broker.PublishOption {
	return broker.PublishContextWithValue(keysKey{}, keys)
}

func WithShardingKey(key string) broker.PublishOption {
	return broker.PublishContextWithValue(shardingKeyKey{}, key)
}
