package kafka

import (
	kafkago "github.com/segmentio/kafka-go"
	"github.com/tx7do/kratos-transport/broker"
	"time"
)

///////////////////////////////////////////////////////////////////////////////

type readerConfigKey struct{}
type retriesCountKey struct{}

func WithReaderConfig(c kafkago.ReaderConfig) broker.Option {
	return broker.OptionContextWithValue(readerConfigKey{}, c)
}

func WithRetriesCount(cnt int) broker.Option {
	return broker.OptionContextWithValue(retriesCountKey{}, cnt)
}

///////////////////////////////////////////////////////////////////////////////

type headersKey struct{}
type batchSizeKey struct{}
type batchTimeoutKey struct{}
type batchBytesKey struct{}
type asyncKey struct{}

func WithHeaders(h map[string]interface{}) broker.PublishOption {
	return broker.PublishContextWithValue(headersKey{}, h)
}

func WithBatchSize(n int) broker.PublishOption {
	return broker.PublishContextWithValue(batchSizeKey{}, n)
}

func WithBatchTimeout(tm time.Duration) broker.PublishOption {
	return broker.PublishContextWithValue(batchTimeoutKey{}, tm)
}

func WithBatchBytes(by int64) broker.PublishOption {
	return broker.PublishContextWithValue(batchBytesKey{}, by)
}

func WithAsync(enable bool) broker.PublishOption {
	return broker.PublishContextWithValue(asyncKey{}, enable)
}
