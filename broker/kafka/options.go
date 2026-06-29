package kafka

import (
	"context"
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

type readerConfigKey struct{}
type writerConfigKey struct{}
type retriesCountKey struct{}
type mechanismKey struct{}
type loggerKey struct{}
type errorLoggerKey struct{}
type enableLoggerKey struct{}
type enableErrorLoggerKey struct{}
type enableOneTopicOneWriterKey struct{}
type batchSizeKey struct{}
type batchTimeoutKey struct{}
type batchBytesKey struct{}
type asyncKey struct{}
type readTimeoutKey struct{}
type writeTimeoutKey struct{}
type allowPublishAutoTopicCreationKey struct{}
type completionKey struct{}

// WithReaderConfig .
func WithReaderConfig(cfg kafkaGo.ReaderConfig) broker.Option {
	return broker.OptionContextWithValue(readerConfigKey{}, cfg)
}

// WithWriterConfig .
func WithWriterConfig(cfg WriterConfig) broker.Option {
	return broker.OptionContextWithValue(writerConfigKey{}, cfg)
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
		LogErrorf("create SCRAM mechanism failed: %v", err)
		return func(o *broker.Options) {}
	}

	return broker.OptionContextWithValue(mechanismKey{}, mechanism)
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

// WithCompletion 消息发布完成回调
func WithCompletion(completion func(messages []kafkaGo.Message, err error)) broker.Option {
	return broker.OptionContextWithValue(completionKey{}, completion)
}

///
/// PublishOption
///

type balancerKey struct{}
type balancerValue struct {
	Name       BalancerName
	Consistent bool
	Hasher     hash.Hash32
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
type dialerConfigKey struct{}
type dialerTimeoutKey struct{}
type maxAttemptsKey struct{}
type partitionKey struct{}
type readBatchTimeoutKey struct{}
type readBackoffMin struct{}
type readBackoffMax struct{}

type subscribeRetriesCountKey struct{}

type subscribeBatchSizeKey struct{}
type subscribeBatchIntervalKey struct{}

func WithSubscribeAutoCreateTopic(topic string, numPartitions, replicationFactor int) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(autoSubscribeCreateTopicKey{},
		&autoSubscribeCreateTopicValue{
			Topic:             topic,
			NumPartitions:     numPartitions,
			ReplicationFactor: replicationFactor,
		},
	)
}

func GetSubscribeAutoCreateTopic(ctx context.Context) (string, int, int, bool) {
	v, ok := ctx.Value(autoSubscribeCreateTopicKey{}).(*autoSubscribeCreateTopicValue)
	if ok {
		return v.Topic, v.NumPartitions, v.ReplicationFactor, true
	}
	return "", 0, 0, false
}

// WithDialerTimeout .
func WithDialerTimeout(tm time.Duration) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(dialerTimeoutKey{}, tm)
}

func GetDialerTimeout(ctx context.Context) (time.Duration, bool) {
	v, ok := ctx.Value(dialerTimeoutKey{}).(time.Duration)
	return v, ok
}

// WithRetries 设置消息重发的次数
func WithRetries(cnt int) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(subscribeRetriesCountKey{}, cnt)
}

func GetRetries(ctx context.Context) (int, bool) {
	v, ok := ctx.Value(subscribeRetriesCountKey{}).(int)
	return v, ok
}

// WithQueueCapacity .
func WithQueueCapacity(cap int) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(queueCapacityKey{}, cap)
}

func GetQueueCapacity(ctx context.Context) (int, bool) {
	v, ok := ctx.Value(queueCapacityKey{}).(int)
	return v, ok
}

// WithMinBytes fetch.min.bytes
func WithMinBytes(bytes int) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(minBytesKey{}, bytes)
}

func GetMinBytes(ctx context.Context) (int, bool) {
	v, ok := ctx.Value(minBytesKey{}).(int)
	return v, ok
}

// WithMaxBytes .
func WithMaxBytes(bytes int) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(maxBytesKey{}, bytes)
}

func GetMaxBytes(ctx context.Context) (int, bool) {
	v, ok := ctx.Value(maxBytesKey{}).(int)
	return v, ok
}

// WithMaxWait fetch.max.wait.ms
func WithMaxWait(time time.Duration) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(maxWaitKey{}, time)
}

func GetMaxWait(ctx context.Context) (time.Duration, bool) {
	v, ok := ctx.Value(maxWaitKey{}).(time.Duration)
	return v, ok
}

// WithReadLagInterval .
func WithReadLagInterval(interval time.Duration) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(readLagIntervalKey{}, interval)
}

func GetReadLagInterval(ctx context.Context) (time.Duration, bool) {
	v, ok := ctx.Value(readLagIntervalKey{}).(time.Duration)
	return v, ok
}

// WithHeartbeatInterval .
func WithHeartbeatInterval(interval time.Duration) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(heartbeatIntervalKey{}, interval)
}

func GetHeartbeatInterval(ctx context.Context) (time.Duration, bool) {
	v, ok := ctx.Value(heartbeatIntervalKey{}).(time.Duration)
	return v, ok
}

// WithCommitInterval .
func WithCommitInterval(interval time.Duration) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(commitIntervalKey{}, interval)
}

func GetCommitInterval(ctx context.Context) (time.Duration, bool) {
	v, ok := ctx.Value(commitIntervalKey{}).(time.Duration)
	return v, ok
}

// WithPartitionWatchInterval .
func WithPartitionWatchInterval(interval time.Duration) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(partitionWatchIntervalKey{}, interval)
}

func GetPartitionWatchInterval(ctx context.Context) (time.Duration, bool) {
	v, ok := ctx.Value(partitionWatchIntervalKey{}).(time.Duration)
	return v, ok
}

// WithWatchPartitionChanges .
func WithWatchPartitionChanges(enable bool) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(watchPartitionChangesKey{}, enable)
}

func GetWatchPartitionChanges(ctx context.Context) (bool, bool) {
	v, ok := ctx.Value(watchPartitionChangesKey{}).(bool)
	return v, ok
}

// WithSessionTimeout .
func WithSessionTimeout(timeout time.Duration) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(sessionTimeoutKey{}, timeout)
}

func GetSessionTimeout(ctx context.Context) (time.Duration, bool) {
	v, ok := ctx.Value(sessionTimeoutKey{}).(time.Duration)
	return v, ok
}

// WithRebalanceTimeout .
func WithRebalanceTimeout(timeout time.Duration) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(rebalanceTimeoutKey{}, timeout)
}

func GetRebalanceTimeout(ctx context.Context) (time.Duration, bool) {
	v, ok := ctx.Value(rebalanceTimeoutKey{}).(time.Duration)
	return v, ok
}

// WithRetentionTime .
func WithRetentionTime(time time.Duration) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(retentionTimeKey{}, time)
}

func GetRetentionTime(ctx context.Context) (time.Duration, bool) {
	v, ok := ctx.Value(retentionTimeKey{}).(time.Duration)
	return v, ok
}

// WithStartOffset .
func WithStartOffset(offset int64) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(startOffsetKey{}, offset)
}

func GetStartOffset(ctx context.Context) (int64, bool) {
	v, ok := ctx.Value(startOffsetKey{}).(int64)
	return v, ok
}

// WithDialer .
func WithDialer(cfg *kafkaGo.Dialer) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(dialerConfigKey{}, cfg)
}

func GetDialer(ctx context.Context) (*kafkaGo.Dialer, bool) {
	v, ok := ctx.Value(dialerConfigKey{}).(*kafkaGo.Dialer)
	return v, ok
}

func WithPartition(partition int) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(partitionKey{}, partition)
}

func GetPartition(ctx context.Context) (int, bool) {
	v, ok := ctx.Value(partitionKey{}).(int)
	return v, ok
}

func WithReadBatchTimeout(tm time.Duration) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(readBatchTimeoutKey{}, tm)
}

func GetReadBatchTimeout(ctx context.Context) (time.Duration, bool) {
	v, ok := ctx.Value(readBatchTimeoutKey{}).(time.Duration)
	return v, ok
}

func WithReadBackoffMin(tm time.Duration) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(readBackoffMin{}, tm)
}

func GetReadBackoffMin(ctx context.Context) (time.Duration, bool) {
	v, ok := ctx.Value(readBackoffMin{}).(time.Duration)
	return v, ok
}

func WithReadBackoffMax(tm time.Duration) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(readBackoffMax{}, tm)
}

func GetReadBackoffMax(ctx context.Context) (time.Duration, bool) {
	v, ok := ctx.Value(readBackoffMax{}).(time.Duration)
	return v, ok
}

func WithSubscribeBatchSize(batchSize int) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(subscribeBatchSizeKey{}, batchSize)
}

func GetSubscribeBatchSize(ctx context.Context) (int, bool) {
	v, ok := ctx.Value(subscribeBatchSizeKey{}).(int)
	return v, ok
}

func WithSubscribeBatchInterval(batchInterval time.Duration) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(subscribeBatchIntervalKey{}, batchInterval)
}

func GetSubscribeBatchInterval(ctx context.Context) (time.Duration, bool) {
	v, ok := ctx.Value(subscribeBatchIntervalKey{}).(time.Duration)
	return v, ok
}
