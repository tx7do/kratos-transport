package rabbitmq

import (
	"context"
	"time"

	"github.com/tx7do/kratos-transport/broker"
)

///
/// Option
///

type durableExchangeKey struct{}
type exchangeKey struct{}
type prefetchCountKey struct{}
type prefetchGlobalKey struct{}
type externalAuthKey struct{}

func WithDurableExchange() broker.Option {
	return broker.OptionContextWithValue(durableExchangeKey{}, true)
}

func WithExchangeName(e string) broker.Option {
	return broker.OptionContextWithValue(exchangeKey{}, e)
}

func WithPrefetchCount(c int) broker.Option {
	return broker.OptionContextWithValue(prefetchCountKey{}, c)
}

func WithPrefetchGlobal() broker.Option {
	return broker.OptionContextWithValue(prefetchGlobalKey{}, true)
}

func WithExternalAuth() broker.Option {
	return broker.OptionContextWithValue(externalAuthKey{}, ExternalAuthentication{})
}

///
/// SubscribeOption
///

type durableQueueKey struct{}
type subscribeHeadersKey struct{}
type queueArgumentsKey struct{}
type requeueOnErrorKey struct{}
type subscribeContextKey struct{}
type ackSuccessKey struct{}

func WithDurableQueue() broker.SubscribeOption {
	return broker.SubscribeContextWithValue(durableQueueKey{}, true)
}

func WithSubscribeHeaders(h map[string]interface{}) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(subscribeHeadersKey{}, h)
}

func WithQueueArguments(h map[string]interface{}) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(queueArgumentsKey{}, h)
}

func WithRequeueOnError() broker.SubscribeOption {
	return broker.SubscribeContextWithValue(requeueOnErrorKey{}, true)
}

func WithSubscribeContext(ctx context.Context) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(subscribeContextKey{}, ctx)
}

func WithAckOnSuccess() broker.SubscribeOption {
	return broker.SubscribeContextWithValue(ackSuccessKey{}, true)
}

///
/// PublishOption
///

type deliveryModeKey struct{}
type priorityKey struct{}
type contentTypeKey struct{}
type contentEncodingKey struct{}
type correlationIDKey struct{}
type replyToKey struct{}
type expirationKey struct{}
type messageIDKey struct{}
type timestampKey struct{}
type messageTypeKey struct{}
type userIDKey struct{}
type appIDKey struct{}
type publishHeadersKey struct{}

func WithDeliveryMode(value uint8) broker.PublishOption {
	return broker.PublishContextWithValue(deliveryModeKey{}, value)
}

func WithPriority(value uint8) broker.PublishOption {
	return broker.PublishContextWithValue(priorityKey{}, value)
}

func WithContentType(value string) broker.PublishOption {
	return broker.PublishContextWithValue(contentTypeKey{}, value)
}

func WithContentEncoding(value string) broker.PublishOption {
	return broker.PublishContextWithValue(contentEncodingKey{}, value)
}

func WithCorrelationID(value string) broker.PublishOption {
	return broker.PublishContextWithValue(correlationIDKey{}, value)
}

func WithReplyTo(value string) broker.PublishOption {
	return broker.PublishContextWithValue(replyToKey{}, value)
}

func WithExpiration(value string) broker.PublishOption {
	return broker.PublishContextWithValue(expirationKey{}, value)
}

func WithMessageId(value string) broker.PublishOption {
	return broker.PublishContextWithValue(messageIDKey{}, value)
}

func WithTimestamp(value time.Time) broker.PublishOption {
	return broker.PublishContextWithValue(timestampKey{}, value)
}

func WithTypeMsg(value string) broker.PublishOption {
	return broker.PublishContextWithValue(messageTypeKey{}, value)
}

func WithUserID(value string) broker.PublishOption {
	return broker.PublishContextWithValue(userIDKey{}, value)
}

func WithAppID(value string) broker.PublishOption {
	return broker.PublishContextWithValue(appIDKey{}, value)
}

func WithPublishHeaders(h map[string]interface{}) broker.PublishOption {
	return broker.PublishContextWithValue(publishHeadersKey{}, h)
}
