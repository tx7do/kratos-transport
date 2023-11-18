package kafka

import (
	"hash"
	"time"

	kafkaGo "github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"

	"github.com/tx7do/kratos-transport/broker"
)

type BalancerName string

const (
	LeastBytesBalancer    BalancerName = "LeastBytes"
	RoundRobinBalancer    BalancerName = "RoundRobin"
	HashBalancer          BalancerName = "Hash"
	ReferenceHashBalancer BalancerName = "ReferenceHash"
	Crc32Balancer         BalancerName = "CRC32Balancer"
	Murmur2Balancer       BalancerName = "Murmur2Balancer"
)

type ScramAlgorithm string

const (
	ScramAlgorithmSHA256 ScramAlgorithm = "SHA256"
	ScramAlgorithmSHA512 ScramAlgorithm = "SHA512"
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
type writerConfigKey struct{}
type dialerConfigKey struct{}
type dialerTimeoutKey struct{}
type loggerKey struct{}
type errorLoggerKey struct{}
type enableLoggerKey struct{}
type enableErrorLoggerKey struct{}
type enableOneTopicOneWriterKey struct{}
type batchSizeKey struct{}
type batchTimeoutKey struct{}
type batchBytesKey struct{}
type asyncKey struct{}
type maxAttemptsKey struct{}
type readTimeoutKey struct{}
type writeTimeoutKey struct{}
type allowPublishAutoTopicCreationKey struct{}

// WithReaderConfig .
func WithReaderConfig(cfg kafkaGo.ReaderConfig) broker.Option {
	return broker.OptionContextWithValue(readerConfigKey{}, cfg)
}

// WithWriterConfig .
func WithWriterConfig(cfg WriterConfig) broker.Option {
	return broker.OptionContextWithValue(writerConfigKey{}, cfg)
}

// WithDialer .
func WithDialer(cfg *kafkaGo.Dialer) broker.Option {
	return broker.OptionContextWithValue(dialerConfigKey{}, cfg)
}

// WithEnableOneTopicOneWriter .
func WithEnableOneTopicOneWriter(enable bool) broker.Option {
	return broker.OptionContextWithValue(enableOneTopicOneWriterKey{}, enable)
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
func WithScramMechanism(algoName ScramAlgorithm, username, password string) broker.Option {
	var algo scram.Algorithm
	switch algoName {
	case ScramAlgorithmSHA256:
		algo = scram.SHA256
	case ScramAlgorithmSHA512:
		algo = scram.SHA512
	}

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

// WithMinBytes fetch.min.bytes
func WithMinBytes(bytes int) broker.Option {
	return broker.OptionContextWithValue(minBytesKey{}, bytes)
}

// WithMaxBytes .
func WithMaxBytes(bytes int) broker.Option {
	return broker.OptionContextWithValue(maxBytesKey{}, bytes)
}

// WithMaxWait fetch.max.wait.ms
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

// WithLogger inject info logger
func WithLogger(l kafkaGo.Logger) broker.Option {
	return broker.OptionContextWithValue(loggerKey{}, l)
}

// WithErrorLogger inject error logger
func WithErrorLogger(l kafkaGo.Logger) broker.Option {
	return broker.OptionContextWithValue(errorLoggerKey{}, l)
}

// WithEnableLogger enable kratos info logger
func WithEnableLogger(enable bool) broker.Option {
	return broker.OptionContextWithValue(enableLoggerKey{}, enable)
}

// WithEnableErrorLogger enable kratos error logger
func WithEnableErrorLogger(enable bool) broker.Option {
	return broker.OptionContextWithValue(enableErrorLoggerKey{}, enable)
}

// WithBatchSize 发送批次大小 batch.size
//
//	default：100
func WithBatchSize(size int) broker.Option {
	return broker.OptionContextWithValue(batchSizeKey{}, size)
}

// WithBatchTimeout linger.ms
//
// default：10ms
func WithBatchTimeout(timeout time.Duration) broker.Option {
	return broker.OptionContextWithValue(batchTimeoutKey{}, timeout)
}

// WithBatchBytes
//
// default：1048576 bytes
func WithBatchBytes(by int64) broker.Option {
	return broker.OptionContextWithValue(batchBytesKey{}, by)
}

// WithAsync 异步发送消息 default：true
func WithAsync(enable bool) broker.Option {
	return broker.OptionContextWithValue(asyncKey{}, enable)
}

// WithPublishMaxAttempts .
func WithPublishMaxAttempts(cnt int) broker.Option {
	return broker.OptionContextWithValue(maxAttemptsKey{}, cnt)
}

// WithReadTimeout 读取超时时间 default：10s
func WithReadTimeout(timeout time.Duration) broker.Option {
	return broker.OptionContextWithValue(readTimeoutKey{}, timeout)
}

// WithWriteTimeout 写入超时时间 default：10s
func WithWriteTimeout(timeout time.Duration) broker.Option {
	return broker.OptionContextWithValue(writeTimeoutKey{}, timeout)
}

// WithAllowPublishAutoTopicCreation .
func WithAllowPublishAutoTopicCreation(enable bool) broker.Option {
	return broker.OptionContextWithValue(allowPublishAutoTopicCreationKey{}, enable)
}

///
/// PublishOption
///

type messageHeadersKey struct{}
type messageKeyKey struct{}
type messageOffsetKey struct{}
type balancerKey struct{}
type balancerValue struct {
	Name       BalancerName
	Consistent bool
	Hasher     hash.Hash32
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

// WithLeastBytesBalancer LeastBytes负载均衡器
func WithLeastBytesBalancer() broker.PublishOption {
	return broker.PublishContextWithValue(balancerKey{},
		&balancerValue{
			Name: LeastBytesBalancer,
		},
	)
}

// WithRoundRobinBalancer RoundRobin负载均衡器，默认均衡器。
func WithRoundRobinBalancer() broker.PublishOption {
	return broker.PublishContextWithValue(balancerKey{},
		&balancerValue{
			Name: RoundRobinBalancer,
		},
	)
}

// WithHashBalancer Hash负载均衡器
func WithHashBalancer(hasher hash.Hash32) broker.PublishOption {
	return broker.PublishContextWithValue(balancerKey{},
		&balancerValue{
			Name:   HashBalancer,
			Hasher: hasher,
		},
	)
}

// WithReferenceHashBalancer ReferenceHash负载均衡器
func WithReferenceHashBalancer(hasher hash.Hash32) broker.PublishOption {
	return broker.PublishContextWithValue(balancerKey{},
		&balancerValue{
			Name:   ReferenceHashBalancer,
			Hasher: hasher,
		},
	)
}

// WithCrc32Balancer CRC32负载均衡器
func WithCrc32Balancer(consistent bool) broker.PublishOption {
	return broker.PublishContextWithValue(balancerKey{},
		&balancerValue{
			Name:       Crc32Balancer,
			Consistent: consistent,
		},
	)
}

// WithMurmur2Balancer Murmur2负载均衡器
func WithMurmur2Balancer(consistent bool) broker.PublishOption {
	return broker.PublishContextWithValue(balancerKey{},
		&balancerValue{
			Name:       Murmur2Balancer,
			Consistent: consistent,
		},
	)
}

///
/// SubscribeOption
///

type autoSubscribeCreateTopicKey struct{}
type autoSubscribeCreateTopicValue struct {
	Topic             string
	NumPartitions     int
	ReplicationFactor int
}

func WithSubscribeAutoCreateTopic(topic string, numPartitions, replicationFactor int) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(autoSubscribeCreateTopicKey{},
		&autoSubscribeCreateTopicValue{
			Topic:             topic,
			NumPartitions:     numPartitions,
			ReplicationFactor: replicationFactor,
		},
	)
}
