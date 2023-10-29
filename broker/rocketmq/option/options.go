package rocketmqOption

import (
	"time"

	rmqClient "github.com/apache/rocketmq-clients/golang/v5"
	"github.com/tx7do/kratos-transport/broker"
)

///
/// Option
///

type Credentials struct {
	AccessKey, AccessSecret, SecurityToken string
}

func WithEnableTrace() broker.Option {
	return broker.OptionContextWithValue(EnableTraceKey{}, true)
}

func WithNameServer(addrs []string) broker.Option {
	return broker.OptionContextWithValue(NameServersKey{}, addrs)
}
func WithNameServerDomain(uri string) broker.Option {
	return broker.OptionContextWithValue(NameServerUrlKey{}, uri)
}

func WithAccessKey(key string) broker.Option {
	return broker.OptionContextWithValue(AccessKey{}, key)
}
func WithSecretKey(secret string) broker.Option {
	return broker.OptionContextWithValue(SecretKey{}, secret)
}
func WithSecurityToken(token string) broker.Option {
	return broker.OptionContextWithValue(SecurityTokenKey{}, token)
}
func WithCredentials(accessKey, accessSecret, securityToken string) broker.Option {
	return broker.OptionContextWithValue(CredentialsKey{},
		&Credentials{
			AccessKey:     accessKey,
			AccessSecret:  accessSecret,
			SecurityToken: securityToken,
		},
	)
}

func WithRetryCount(count int) broker.Option {
	return broker.OptionContextWithValue(RetryCountKey{}, count)
}

func WithNamespace(ns string) broker.Option {
	return broker.OptionContextWithValue(NamespaceKey{}, ns)
}

func WithInstanceName(name string) broker.Option {
	return broker.OptionContextWithValue(InstanceNameKey{}, name)
}

func WithGroupName(name string) broker.Option {
	return broker.OptionContextWithValue(GroupNameKey{}, name)
}

func WithSubscriptionExpressions(subscriptionExpressions map[string]*rmqClient.FilterExpression) broker.Option {
	return broker.OptionContextWithValue(SubscriptionExpressionsKey{}, subscriptionExpressions)
}

func WithAwaitDuration(awaitDuration time.Duration) broker.Option {
	return broker.OptionContextWithValue(AwaitDurationKey{}, awaitDuration)
}

func WithMaxMessageNumKey(messageNum int32) broker.Option {
	return broker.OptionContextWithValue(MaxMessageNumKey{}, messageNum)
}

func WithInvisibleDuration(invisibleDuration time.Duration) broker.Option {
	return broker.OptionContextWithValue(InvisibleDurationKey{}, invisibleDuration)
}

func WithReceiveInterval(receiveInterval time.Duration) broker.Option {
	return broker.OptionContextWithValue(ReceiveIntervalKey{}, receiveInterval)
}

///
/// PublishOption
///

func WithCompress(compress bool) broker.PublishOption {
	return broker.PublishContextWithValue(CompressKey{}, compress)
}

func WithBatch(batch bool) broker.PublishOption {
	return broker.PublishContextWithValue(BatchKey{}, batch)
}

func WithProperties(properties map[string]string) broker.PublishOption {
	return broker.PublishContextWithValue(PropertiesKey{}, properties)
}

func WithDelayTimeLevel(level int) broker.PublishOption {
	return broker.PublishContextWithValue(DelayTimeLevelKey{}, level)
}

func WithTag(tags string) broker.PublishOption {
	return broker.PublishContextWithValue(TagsKey{}, tags)
}

func WithKeys(keys []string) broker.PublishOption {
	return broker.PublishContextWithValue(KeysKey{}, keys)
}

func WithShardingKey(key string) broker.PublishOption {
	return broker.PublishContextWithValue(ShardingKeyKey{}, key)
}

func WithDeliveryTimestamp(deliveryTimestamp time.Time) broker.PublishOption {
	return broker.PublishContextWithValue(DeliveryTimestampKey{}, deliveryTimestamp)
}

func WithMessageGroup(group string) broker.PublishOption {
	return broker.PublishContextWithValue(MessageGroupKey{}, group)
}

///
/// SubscribeOption
///

func WithSubscriptionFilterExpression(filterExpression *rmqClient.FilterExpression) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(SubscriptionFilterExpressionKey{}, filterExpression)
}
