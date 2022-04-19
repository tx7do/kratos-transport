package rabbitmq

import (
	"context"
	"time"

	"github.com/tx7do/kratos-transport/broker"
)

type durableQueueKey struct{}
type headersKey struct{}
type queueArgumentsKey struct{}
type prefetchCountKey struct{}
type prefetchGlobalKey struct{}
type exchangeKey struct{}
type requeueOnErrorKey struct{}
type deliveryModeKey struct{}
type priorityKey struct{}
type ackSuccessKey struct{}
type contentTypeKey struct{}
type contentEncodingKey struct{}
type correlationIDKey struct{}
type replyToKey struct{}
type expirationKey struct{}
type messageIDKey struct{}
type timestampKey struct{}
type typeMsgKey struct{}
type userIDKey struct{}
type appIDKey struct{}
type externalAuthKey struct{}
type durableExchangeKey struct{}
type subscribeContextKey struct{}

///////////////////////////////////////////////////////////////////////////////

func DurableExchange() broker.Option {
	return broker.OptionContextWithValue(durableExchangeKey{}, true)
}

func ExchangeName(e string) broker.Option {
	return broker.OptionContextWithValue(exchangeKey{}, e)
}

func PrefetchCount(c int) broker.Option {
	return broker.OptionContextWithValue(prefetchCountKey{}, c)
}

func PrefetchGlobal() broker.Option {
	return broker.OptionContextWithValue(prefetchGlobalKey{}, true)
}

func ExternalAuth() broker.Option {
	return broker.OptionContextWithValue(externalAuthKey{}, ExternalAuthentication{})
}

///////////////////////////////////////////////////////////////////////////////

func DurableQueue() broker.SubscribeOption {
	return broker.SubscribeContextWithValue(durableQueueKey{}, true)
}

func Headers(h map[string]interface{}) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(headersKey{}, h)
}

func QueueArguments(h map[string]interface{}) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(queueArgumentsKey{}, h)
}

func RequeueOnError() broker.SubscribeOption {
	return broker.SubscribeContextWithValue(requeueOnErrorKey{}, true)
}

func SubscribeContext(ctx context.Context) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(subscribeContextKey{}, ctx)
}

func SubscribeContextFromContext(ctx context.Context) (context.Context, bool) {
	c, ok := ctx.Value(subscribeContextKey{}).(context.Context)
	return c, ok
}

func AckOnSuccess() broker.SubscribeOption {
	return broker.SubscribeContextWithValue(ackSuccessKey{}, true)
}

func AckOnSuccessFromContext(ctx context.Context) (bool, bool) {
	b, ok := ctx.Value(ackSuccessKey{}).(bool)
	return b, ok
}

///////////////////////////////////////////////////////////////////////////////

func DeliveryMode(value uint8) broker.PublishOption {
	return broker.PublishContextWithValue(deliveryModeKey{}, value)
}

func Priority(value uint8) broker.PublishOption {
	return broker.PublishContextWithValue(priorityKey{}, value)
}

func ContentType(value string) broker.PublishOption {
	return broker.PublishContextWithValue(contentTypeKey{}, value)
}

func ContentEncoding(value string) broker.PublishOption {
	return broker.PublishContextWithValue(contentEncodingKey{}, value)
}

func CorrelationID(value string) broker.PublishOption {
	return broker.PublishContextWithValue(correlationIDKey{}, value)
}

func ReplyTo(value string) broker.PublishOption {
	return broker.PublishContextWithValue(replyToKey{}, value)
}

func Expiration(value string) broker.PublishOption {
	return broker.PublishContextWithValue(expirationKey{}, value)
}

func MessageId(value string) broker.PublishOption {
	return broker.PublishContextWithValue(messageIDKey{}, value)
}

func Timestamp(value time.Time) broker.PublishOption {
	return broker.PublishContextWithValue(timestampKey{}, value)
}

func TypeMsg(value string) broker.PublishOption {
	return broker.PublishContextWithValue(typeMsgKey{}, value)
}

func UserID(value string) broker.PublishOption {
	return broker.PublishContextWithValue(userIDKey{}, value)
}

func AppID(value string) broker.PublishOption {
	return broker.PublishContextWithValue(appIDKey{}, value)
}
