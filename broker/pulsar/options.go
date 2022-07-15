package pulsar

import (
	"github.com/tx7do/kratos-transport/broker"
	"time"
)

///
/// Option
///

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
type deliverAfterKey struct{}
type deliverAtKey struct{}

func WithProducerNamePublish(name string) broker.PublishOption {
	return broker.PublishContextWithValue(producerNameKey{}, name)
}

func WithProducerPropertiesPublish(properties map[string]string) broker.PublishOption {
	return broker.PublishContextWithValue(producerPropertiesKey{}, properties)
}

func WithSendTimeoutPublish(timeout time.Duration) broker.PublishOption {
	return broker.PublishContextWithValue(sendTimeoutKey{}, timeout)
}

func WithDisableBatchingPublish(disable bool) broker.PublishOption {
	return broker.PublishContextWithValue(disableBatchingKey{}, disable)
}

func WithBatchingMaxPublishDelayPublish(delay time.Duration) broker.PublishOption {
	return broker.PublishContextWithValue(batchingMaxPublishDelayKey{}, delay)
}

func WithBatchingMaxMessagesPublish(size uint) broker.PublishOption {
	return broker.PublishContextWithValue(batchingMaxMessagesKey{}, size)
}

func WithBatchingMaxSizePublish(size uint) broker.PublishOption {
	return broker.PublishContextWithValue(batchingMaxSizeKey{}, size)
}

func WithDeliverAfterPublish(delay time.Duration) broker.PublishOption {
	return broker.PublishContextWithValue(deliverAfterKey{}, delay)
}

func WithDeliverAtPublish(tm time.Time) broker.PublishOption {
	return broker.PublishContextWithValue(deliverAtKey{}, tm)
}

///
/// SubscribeOption
///
