package rabbitmq

import (
	"context"
	"time"

	"github.com/tx7do/kratos-transport/broker"
)

///
/// Option
///

type exchangeDurableKey struct{}
type exchangeNameKey struct{}
type exchangeKindKey struct{}

type prefetchCountKey struct{}
type prefetchSizeKey struct{}
type prefetchGlobalKey struct{}
type externalAuthKey struct{}

// WithDurableExchange Exchange.Durable
func WithDurableExchange() broker.Option {
	return broker.OptionContextWithValue(exchangeDurableKey{}, true)
}

// WithExchangeName Exchange.Name
func WithExchangeName(name string) broker.Option {
	return broker.OptionContextWithValue(exchangeNameKey{}, name)
}

// WithExchangeType Exchange.Type
func WithExchangeType(kind string) broker.Option {
	return broker.OptionContextWithValue(exchangeKindKey{}, kind)
}

// WithPrefetchCount Channel.Qos.PrefetchCount
func WithPrefetchCount(cnt int) broker.Option {
	return broker.OptionContextWithValue(prefetchCountKey{}, cnt)
}

// WithPrefetchSize Channel.Qos.PrefetchSize
func WithPrefetchSize(size int) broker.Option {
	return broker.OptionContextWithValue(prefetchSizeKey{}, size)
}

// WithPrefetchGlobal Channel.Qos.Global
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
type subscribeBindArgsKey struct{}
type subscribeQueueArgsKey struct{}
type requeueOnErrorKey struct{}
type subscribeContextKey struct{}
type ackSuccessKey struct{}
type autoDeleteQueueKey struct{}

func WithDurableQueue() broker.SubscribeOption {
	return broker.SubscribeContextWithValue(durableQueueKey{}, true)
}

func WithAutoDeleteQueue() broker.SubscribeOption {
	return broker.SubscribeContextWithValue(autoDeleteQueueKey{}, true)
}

func WithBindArguments(args map[string]interface{}) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(subscribeBindArgsKey{}, args)
}

func WithQueueArguments(args map[string]interface{}) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(subscribeQueueArgsKey{}, args)
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

type DeclarePublishQueueInfo struct {
	QueueArguments map[string]interface{}
	BindArguments  map[string]interface{}
	Durable        bool
	AutoDelete     bool
	Queue          string
}

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
type publishDeclareQueueKey struct{}

// WithDeliveryMode amqp.Publishing.DeliveryMode
func WithDeliveryMode(value uint8) broker.PublishOption {
	return broker.PublishContextWithValue(deliveryModeKey{}, value)
}

// WithPriority amqp.Publishing.Priority
func WithPriority(value uint8) broker.PublishOption {
	return broker.PublishContextWithValue(priorityKey{}, value)
}

// WithContentType amqp.Publishing.ContentType
func WithContentType(value string) broker.PublishOption {
	return broker.PublishContextWithValue(contentTypeKey{}, value)
}

// WithContentEncoding amqp.Publishing.ContentEncoding
func WithContentEncoding(value string) broker.PublishOption {
	return broker.PublishContextWithValue(contentEncodingKey{}, value)
}

// WithCorrelationID amqp.Publishing.CorrelationId
func WithCorrelationID(value string) broker.PublishOption {
	return broker.PublishContextWithValue(correlationIDKey{}, value)
}

// WithReplyTo amqp.Publishing.ReplyTo
func WithReplyTo(value string) broker.PublishOption {
	return broker.PublishContextWithValue(replyToKey{}, value)
}

// WithExpiration amqp.Publishing.Expiration
func WithExpiration(value string) broker.PublishOption {
	return broker.PublishContextWithValue(expirationKey{}, value)
}

// WithMessageId amqp.Publishing.MessageId
func WithMessageId(value string) broker.PublishOption {
	return broker.PublishContextWithValue(messageIDKey{}, value)
}

// WithTimestamp amqp.Publishing.Timestamp
func WithTimestamp(value time.Time) broker.PublishOption {
	return broker.PublishContextWithValue(timestampKey{}, value)
}

// WithTypeMsg amqp.Publishing.Type
func WithTypeMsg(value string) broker.PublishOption {
	return broker.PublishContextWithValue(messageTypeKey{}, value)
}

// WithUserID amqp.Publishing.UserId
func WithUserID(value string) broker.PublishOption {
	return broker.PublishContextWithValue(userIDKey{}, value)
}

// WithAppID amqp.Publishing.AppId
func WithAppID(value string) broker.PublishOption {
	return broker.PublishContextWithValue(appIDKey{}, value)
}

// WithPublishHeaders amqp.Publishing.Headers
func WithPublishHeaders(h map[string]interface{}) broker.PublishOption {
	return broker.PublishContextWithValue(publishHeadersKey{}, h)
}

// WithPublishDeclareQueue publish declare queue info
func WithPublishDeclareQueue(queueName string, durableQueue, autoDelete bool, queueArgs map[string]interface{}, bindArgs map[string]interface{}) broker.PublishOption {
	val := &DeclarePublishQueueInfo{
		Queue:          queueName,
		Durable:        durableQueue,
		AutoDelete:     autoDelete,
		QueueArguments: queueArgs,
		BindArguments:  bindArgs,
	}
	return broker.PublishContextWithValue(publishDeclareQueueKey{}, val)
}
