package pulsar

import (
	"github.com/tx7do/kratos-transport/broker"
	"time"
)

///
/// Option
///

type connectionTimeoutKey struct{}
type operationTimeoutKey struct{}
type listenerNameKey struct{}
type maxConnectionsPerBrokerKey struct{}
type customMetricsLabelsKey struct{}
type tlsKey struct{}

// WithConnectionTimeout ClientOptions.ConnectionTimeout
func WithConnectionTimeout(timeout time.Duration) broker.Option {
	return broker.OptionContextWithValue(connectionTimeoutKey{}, timeout)
}

// WithOperationTimeout ClientOptions.ConnectionTimeout
func WithOperationTimeout(timeout time.Duration) broker.Option {
	return broker.OptionContextWithValue(operationTimeoutKey{}, timeout)
}

// WithListenerName ClientOptions.ConnectionTimeout
func WithListenerName(name string) broker.Option {
	return broker.OptionContextWithValue(listenerNameKey{}, name)
}

// WithMaxConnectionsPerBroker ClientOptions.ConnectionTimeout
func WithMaxConnectionsPerBroker(cnt int) broker.Option {
	return broker.OptionContextWithValue(maxConnectionsPerBrokerKey{}, cnt)
}

// WithCustomMetricsLabels ClientOptions.ConnectionTimeout
func WithCustomMetricsLabels(labels map[string]string) broker.Option {
	return broker.OptionContextWithValue(customMetricsLabelsKey{}, labels)
}

type tlsConfig struct {
	ClientCertPath          string
	ClientKeyPath           string
	CaCertsPath             string
	AllowInsecureConnection bool
	ValidateHostname        bool
}

// WithTLSConfig set tls config for client
func WithTLSConfig(caCertsPath, tlsClientCertPath, tlsClientKeyPath string, allowInsecureConnection, validateHostname bool) broker.Option {
	cfg := tlsConfig{
		tlsClientCertPath,
		tlsClientKeyPath,
		caCertsPath,
		allowInsecureConnection,
		validateHostname,
	}
	return broker.OptionContextWithValue(tlsKey{}, cfg)
}

///
/// PublishOption
///

type producerNameKey struct{}
type producerPropertiesKey struct{}
type sendTimeoutKey struct{}
type disableBatchingKey struct{}
type batchingMaxPublishDelayKey struct{}
type batchingMaxMessagesKey struct{}
type batchingMaxSizeKey struct{}

type messageDeliverAfterKey struct{}
type messageDeliverAtKey struct{}
type messageHeadersKey struct{}
type messageSequenceIdKey struct{}
type messageKeyKey struct{}
type messageValueKey struct{}
type messageOrderingKeyKey struct{}
type messageEventTimeKey struct{}
type messageDisableReplication struct{}

// WithProducerName ProducerOptions.Name
func WithProducerName(name string) broker.PublishOption {
	return broker.PublishContextWithValue(producerNameKey{}, name)
}

// WithProducerProperties ProducerOptions.Properties
func WithProducerProperties(properties map[string]string) broker.PublishOption {
	return broker.PublishContextWithValue(producerPropertiesKey{}, properties)
}

// WithSendTimeout ProducerOptions.SendTimeout
func WithSendTimeout(timeout time.Duration) broker.PublishOption {
	return broker.PublishContextWithValue(sendTimeoutKey{}, timeout)
}

// WithDisableBatching ProducerOptions.DisableBatching
func WithDisableBatching(disable bool) broker.PublishOption {
	return broker.PublishContextWithValue(disableBatchingKey{}, disable)
}

// WithBatchingMaxPublishDelay ProducerOptions.BatchingMaxPublishDelay
func WithBatchingMaxPublishDelay(delay time.Duration) broker.PublishOption {
	return broker.PublishContextWithValue(batchingMaxPublishDelayKey{}, delay)
}

// WithBatchingMaxMessages ProducerOptions.BatchingMaxMessages
func WithBatchingMaxMessages(size uint) broker.PublishOption {
	return broker.PublishContextWithValue(batchingMaxMessagesKey{}, size)
}

// WithBatchingMaxSize ProducerOptions.BatchingMaxSize
func WithBatchingMaxSize(size uint) broker.PublishOption {
	return broker.PublishContextWithValue(batchingMaxSizeKey{}, size)
}

// WithDeliverAfter ProducerMessage.DeliverAfter
func WithDeliverAfter(delay time.Duration) broker.PublishOption {
	return broker.PublishContextWithValue(messageDeliverAfterKey{}, delay)
}

// WithDeliverAt ProducerMessage.DeliverAt
func WithDeliverAt(tm time.Time) broker.PublishOption {
	return broker.PublishContextWithValue(messageDeliverAtKey{}, tm)
}

// WithHeaders ProducerMessage.Properties
func WithHeaders(headers map[string]string) broker.PublishOption {
	return broker.PublishContextWithValue(messageHeadersKey{}, headers)
}

// WithSequenceID ProducerMessage.SequenceID
func WithSequenceID(id *int64) broker.PublishOption {
	return broker.PublishContextWithValue(messageSequenceIdKey{}, id)
}

// WithMessageKey ProducerMessage.Key
func WithMessageKey(key string) broker.PublishOption {
	return broker.PublishContextWithValue(messageKeyKey{}, key)
}

// WithMessageValue ProducerMessage.Value
func WithMessageValue(value any) broker.PublishOption {
	return broker.PublishContextWithValue(messageValueKey{}, value)
}

// WithMessageOrderingKey ProducerMessage.OrderingKey
func WithMessageOrderingKey(key string) broker.PublishOption {
	return broker.PublishContextWithValue(messageOrderingKeyKey{}, key)
}

// WithMessageEventTime ProducerMessage.EventTime
func WithMessageEventTime(time time.Time) broker.PublishOption {
	return broker.PublishContextWithValue(messageEventTimeKey{}, time)
}

// WithMessageDisableReplication ProducerMessage.DisableReplication
func WithMessageDisableReplication(disable bool) broker.PublishOption {
	return broker.PublishContextWithValue(messageDisableReplication{}, disable)
}

///
/// SubscribeOption
///

type subscriptionNameKey struct{}
type consumerPropertiesKey struct{}
type subscriptionPropertiesKey struct{}
type topicsPatternKey struct{}
type autoDiscoveryPeriodKey struct{}
type nackRedeliveryDelayKey struct{}
type subscriptionRetryEnableKey struct{}
type receiverQueueSizeKey struct{}

// WithSubscriptionName ConsumerOptions.Name
func WithSubscriptionName(name string) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(subscriptionNameKey{}, name)
}

// WithConsumerProperties ConsumerOptions.Properties
func WithConsumerProperties(properties map[string]string) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(consumerPropertiesKey{}, properties)
}

// WithSubscriptionProperties ConsumerOptions.SubscriptionProperties
func WithSubscriptionProperties(properties map[string]string) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(subscriptionPropertiesKey{}, properties)
}

// WithSubscriptionTopicsPattern ConsumerOptions.TopicsPattern
func WithSubscriptionTopicsPattern(pattern string) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(topicsPatternKey{}, pattern)
}

// WithAutoDiscoveryPeriod ConsumerOptions.AutoDiscoveryPeriod
func WithAutoDiscoveryPeriod(period time.Duration) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(autoDiscoveryPeriodKey{}, period)
}

// WithNackRedeliveryDelay ConsumerOptions.NackRedeliveryDelay
func WithNackRedeliveryDelay(delay time.Duration) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(nackRedeliveryDelayKey{}, delay)
}

// WithSubscriptionRetryEnable ConsumerOptions.RetryEnable
func WithSubscriptionRetryEnable(enable bool) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(subscriptionRetryEnableKey{}, enable)
}

// WithReceiverQueueSize ConsumerOptions.ReceiverQueueSize
func WithReceiverQueueSize(size int) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(receiverQueueSizeKey{}, size)
}
