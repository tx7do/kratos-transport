package kafka

import (
	"github.com/tx7do/kratos-transport/broker"
	"time"
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

func WithRetriesCount(cnt int) broker.Option {
	return broker.OptionContextWithValue(retriesCountKey{}, cnt)
}

func WithQueueCapacity(cap int) broker.Option {
	return broker.OptionContextWithValue(queueCapacityKey{}, cap)
}

func WithMinBytes(bytes int) broker.Option {
	return broker.OptionContextWithValue(minBytesKey{}, bytes)
}

func WithMaxBytes(bytes int) broker.Option {
	return broker.OptionContextWithValue(maxBytesKey{}, bytes)
}

func WithMaxWait(time time.Duration) broker.Option {
	return broker.OptionContextWithValue(maxWaitKey{}, time)
}

func WithReadLagInterval(interval time.Duration) broker.Option {
	return broker.OptionContextWithValue(readLagIntervalKey{}, interval)
}

func WithHeartbeatInterval(interval time.Duration) broker.Option {
	return broker.OptionContextWithValue(heartbeatIntervalKey{}, interval)
}

func WithCommitInterval(interval time.Duration) broker.Option {
	return broker.OptionContextWithValue(commitIntervalKey{}, interval)
}

func WithPartitionWatchInterval(interval time.Duration) broker.Option {
	return broker.OptionContextWithValue(partitionWatchIntervalKey{}, interval)
}

func WithWatchPartitionChanges(enable bool) broker.Option {
	return broker.OptionContextWithValue(watchPartitionChangesKey{}, enable)
}

func WithSessionTimeout(timeout time.Duration) broker.Option {
	return broker.OptionContextWithValue(sessionTimeoutKey{}, timeout)
}

func WithRebalanceTimeout(timeout time.Duration) broker.Option {
	return broker.OptionContextWithValue(rebalanceTimeoutKey{}, timeout)
}

func WithRetentionTime(time time.Duration) broker.Option {
	return broker.OptionContextWithValue(retentionTimeKey{}, time)
}

func StartOffset(offset int64) broker.Option {
	return broker.OptionContextWithValue(startOffsetKey{}, offset)
}

func WithMaxAttempts(cnt int) broker.Option {
	return broker.OptionContextWithValue(maxAttemptsKey{}, cnt)
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

func WithBatchSize(size int) broker.PublishOption {
	return broker.PublishContextWithValue(batchSizeKey{}, size)
}

func WithBatchTimeout(timeout time.Duration) broker.PublishOption {
	return broker.PublishContextWithValue(batchTimeoutKey{}, timeout)
}

func WithBatchBytes(by int64) broker.PublishOption {
	return broker.PublishContextWithValue(batchBytesKey{}, by)
}

func WithAsync(enable bool) broker.PublishOption {
	return broker.PublishContextWithValue(asyncKey{}, enable)
}

func WithPublishMaxAttempts(cnt int) broker.PublishOption {
	return broker.PublishContextWithValue(maxAttemptsKey{}, cnt)
}

func WithReadTimeout(timeout time.Duration) broker.PublishOption {
	return broker.PublishContextWithValue(readTimeoutKey{}, timeout)
}

func WithWriteTimeout(timeout time.Duration) broker.PublishOption {
	return broker.PublishContextWithValue(writeTimeoutKey{}, timeout)
}

func WithAllowAutoTopicCreation(enable bool) broker.PublishOption {
	return broker.PublishContextWithValue(allowAutoTopicCreationKey{}, enable)
}

func WithBalancer(balancer string) broker.PublishOption {
	return broker.PublishContextWithValue(balancerKey{}, balancer)
}

func WithHeaders(headers map[string]interface{}) broker.PublishOption {
	return broker.PublishContextWithValue(messageHeadersKey{}, headers)
}

func WithMessage(key []byte) broker.PublishOption {
	return broker.PublishContextWithValue(messageKeyKey{}, key)
}

func WithMessageOffset(offset int64) broker.PublishOption {
	return broker.PublishContextWithValue(messageOffsetKey{}, offset)
}

///
/// SubscribeOption
///
