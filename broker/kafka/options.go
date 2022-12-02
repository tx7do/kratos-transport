package kafka

import (
	"time"

	kafkaGo "github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"

	"github.com/tx7do/kratos-transport/broker"
)

const (
	BalancerLeastBytes      = "LeastBytes"
	BalancerRoundRobin      = "RoundRobin"
	BalancerHash            = "Hash"
	BalancerReferenceHash   = "ReferenceHash"
	BalancerCRC32Balancer   = "CRC32Balancer"
	BalancerMurmur2Balancer = "Murmur2Balancer"
)

///
/// Option
///

type retriesCountKey struct{}
type queueCapacityKey struct{}
type minBytesKey struct{}
type maxBytesKey struct{}
type maxWaitKey struct{}
type readLagIntervalKey struct{}
type heartbeatIntervalKey struct{}
type commitIntervalKey struct{}
type partitionWatchIntervalKey struct{}
type watchPartitionChangesKey struct{}
type sessionTimeoutKey struct{}
type rebalanceTimeoutKey struct{}
type retentionTimeKey struct{}
type startOffsetKey struct{}
type mechanismKey struct{}
type readerConfigKey struct{}
type dialerConfigKey struct{}
type dialerTimeoutKey struct{}
type loggerKey struct{}
type errorLoggerKey struct{}

// WithReaderConfig .
func WithReaderConfig(cfg kafkaGo.ReaderConfig) broker.Option {
	return broker.OptionContextWithValue(readerConfigKey{}, cfg)
}

// WithDialer .
func WithDialer(cfg *kafkaGo.Dialer) broker.Option {
	return broker.OptionContextWithValue(dialerConfigKey{}, cfg)
}

// WithPlainMechanism PLAIN认证信息
func WithPlainMechanism(username, password string) broker.Option {
	mechanism := plain.Mechanism{
		Username: username,
		Password: password,
	}
	return broker.OptionContextWithValue(mechanismKey{}, mechanism)
}

// WithScramMechanism SCRAM认证信息
func WithScramMechanism(algo scram.Algorithm, username, password string) broker.Option {
	mechanism, err := scram.Mechanism(algo, username, password)
	if err != nil {
		panic(err)
	}
	return broker.OptionContextWithValue(mechanismKey{}, mechanism)
}

// WithDialerTimeout .
func WithDialerTimeout(tm time.Duration) broker.Option {
	return broker.OptionContextWithValue(dialerTimeoutKey{}, tm)
}

// WithRetries 设置消息重发的次数
func WithRetries(cnt int) broker.Option {
	return broker.OptionContextWithValue(retriesCountKey{}, cnt)
}

// WithQueueCapacity .
func WithQueueCapacity(cap int) broker.Option {
	return broker.OptionContextWithValue(queueCapacityKey{}, cap)
}

// WithMinBytes .
func WithMinBytes(bytes int) broker.Option {
	return broker.OptionContextWithValue(minBytesKey{}, bytes)
}

// WithMaxBytes .
func WithMaxBytes(bytes int) broker.Option {
	return broker.OptionContextWithValue(maxBytesKey{}, bytes)
}

// WithMaxWait .
func WithMaxWait(time time.Duration) broker.Option {
	return broker.OptionContextWithValue(maxWaitKey{}, time)
}

// WithReadLagInterval .
func WithReadLagInterval(interval time.Duration) broker.Option {
	return broker.OptionContextWithValue(readLagIntervalKey{}, interval)
}

// WithHeartbeatInterval .
func WithHeartbeatInterval(interval time.Duration) broker.Option {
	return broker.OptionContextWithValue(heartbeatIntervalKey{}, interval)
}

// WithCommitInterval .
func WithCommitInterval(interval time.Duration) broker.Option {
	return broker.OptionContextWithValue(commitIntervalKey{}, interval)
}

// WithPartitionWatchInterval .
func WithPartitionWatchInterval(interval time.Duration) broker.Option {
	return broker.OptionContextWithValue(partitionWatchIntervalKey{}, interval)
}

// WithWatchPartitionChanges .
func WithWatchPartitionChanges(enable bool) broker.Option {
	return broker.OptionContextWithValue(watchPartitionChangesKey{}, enable)
}

// WithSessionTimeout .
func WithSessionTimeout(timeout time.Duration) broker.Option {
	return broker.OptionContextWithValue(sessionTimeoutKey{}, timeout)
}

// WithRebalanceTimeout .
func WithRebalanceTimeout(timeout time.Duration) broker.Option {
	return broker.OptionContextWithValue(rebalanceTimeoutKey{}, timeout)
}

// WithRetentionTime .
func WithRetentionTime(time time.Duration) broker.Option {
	return broker.OptionContextWithValue(retentionTimeKey{}, time)
}

// WithStartOffset .
func WithStartOffset(offset int64) broker.Option {
	return broker.OptionContextWithValue(startOffsetKey{}, offset)
}

// WithMaxAttempts .
func WithMaxAttempts(cnt int) broker.Option {
	return broker.OptionContextWithValue(maxAttemptsKey{}, cnt)
}

// WithLogger .
func WithLogger(l kafkaGo.Logger) broker.Option {
	return broker.OptionContextWithValue(loggerKey{}, l)
}

// WithErrorLogger .
func WithErrorLogger(l kafkaGo.Logger) broker.Option {
	return broker.OptionContextWithValue(errorLoggerKey{}, l)
}

///
/// PublishOption
///

type batchSizeKey struct{}
type batchTimeoutKey struct{}
type batchBytesKey struct{}
type asyncKey struct{}
type maxAttemptsKey struct{}
type readTimeoutKey struct{}
type writeTimeoutKey struct{}
type allowAutoTopicCreationKey struct{}
type balancerKey struct{}

type messageHeadersKey struct{}
type messageKeyKey struct{}
type messageOffsetKey struct{}

// WithBatchSize 发送批次大小
//
//	默认：100条
func WithBatchSize(size int) broker.PublishOption {
	return broker.PublishContextWithValue(batchSizeKey{}, size)
}

// WithBatchTimeout 默认：
func WithBatchTimeout(timeout time.Duration) broker.PublishOption {
	return broker.PublishContextWithValue(batchTimeoutKey{}, timeout)
}

// WithBatchBytes
//
//	默认：1048576字节
func WithBatchBytes(by int64) broker.PublishOption {
	return broker.PublishContextWithValue(batchBytesKey{}, by)
}

// WithAsync 异步发送消息
//
//	默认为：false
func WithAsync(enable bool) broker.PublishOption {
	return broker.PublishContextWithValue(asyncKey{}, enable)
}

// WithPublishMaxAttempts .
func WithPublishMaxAttempts(cnt int) broker.PublishOption {
	return broker.PublishContextWithValue(maxAttemptsKey{}, cnt)
}

// WithReadTimeout 读取超时时间
//
//	默认：10秒
func WithReadTimeout(timeout time.Duration) broker.PublishOption {
	return broker.PublishContextWithValue(readTimeoutKey{}, timeout)
}

// WithWriteTimeout 写入超时时间
//
//	默认：10秒
func WithWriteTimeout(timeout time.Duration) broker.PublishOption {
	return broker.PublishContextWithValue(writeTimeoutKey{}, timeout)
}

// WithAllowAutoTopicCreation .
func WithAllowAutoTopicCreation(enable bool) broker.PublishOption {
	return broker.PublishContextWithValue(allowAutoTopicCreationKey{}, enable)
}

// WithBalancer 负载均衡器
//
//	默认： BalancerLeastBytes
//
//	所有支持的均衡器有： BalancerLeastBytes BalancerRoundRobin BalancerHash BalancerReferenceHash BalancerCRC32Balancer BalancerMurmur2Balancer
func WithBalancer(balancer string) broker.PublishOption {
	return broker.PublishContextWithValue(balancerKey{}, balancer)
}

// WithHeaders 消息头
func WithHeaders(headers map[string]interface{}) broker.PublishOption {
	return broker.PublishContextWithValue(messageHeadersKey{}, headers)
}

// WithMessageKey 消息键
func WithMessageKey(key []byte) broker.PublishOption {
	return broker.PublishContextWithValue(messageKeyKey{}, key)
}

// WithMessageOffset 消息偏移
func WithMessageOffset(offset int64) broker.PublishOption {
	return broker.PublishContextWithValue(messageOffsetKey{}, offset)
}

///
/// SubscribeOption
///
