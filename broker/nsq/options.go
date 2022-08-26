package nsq

import (
	"time"

	"github.com/nsqio/go-nsq"
	"github.com/tx7do/kratos-transport/broker"
)

///
/// Option
///

type lookupdAddrsKey struct{}
type consumerOptsKey struct{}

func WithLookupdAddress(addrs []string) broker.Option {
	return broker.OptionContextWithValue(lookupdAddrsKey{}, addrs)
}

func WithConsumerOptions(consumerOpts []string) broker.Option {
	return broker.OptionContextWithValue(consumerOptsKey{}, consumerOpts)
}

///
/// PublishOption
///

type asyncPublishKey struct{}
type deferredPublishKey struct{}

func WithAsyncPublish(doneChan chan *nsq.ProducerTransaction) broker.PublishOption {
	return broker.PublishContextWithValue(asyncPublishKey{}, doneChan)
}

func WithDeferredPublish(delay time.Duration) broker.PublishOption {
	return broker.PublishContextWithValue(deferredPublishKey{}, delay)
}

///
/// SubscribeOption
///

type concurrentHandlerKey struct{}
type maxInFlightKey struct{}

func WithConcurrentHandlers(n int) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(concurrentHandlerKey{}, n)
}

func WithMaxInFlight(n int) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(maxInFlightKey{}, n)
}
