package stomp

import (
	"context"
	"github.com/tx7do/kratos-transport/broker"
	"time"
)

func SubscribeHeaders(h map[string]string) broker.SubscribeOption {
	return setSubscribeOption(subscribeHeaderKey{}, h)
}

type subscribeContextKey struct{}

func SubscribeContext(ctx context.Context) broker.SubscribeOption {
	return setSubscribeOption(subscribeContextKey{}, ctx)
}

type ackSuccessKey struct{}

func AckOnSuccess() broker.SubscribeOption {
	return setSubscribeOption(ackSuccessKey{}, true)
}

func Durable() broker.SubscribeOption {
	return setSubscribeOption(durableQueueKey{}, true)
}

func Receipt(_ time.Duration) broker.PublishOption {
	return setPublishOption(receiptKey{}, true)
}

func SuppressContentLength(_ time.Duration) broker.PublishOption {
	return setPublishOption(suppressContentLengthKey{}, true)
}

func ConnectTimeout(ct time.Duration) broker.Option {
	return setBrokerOption(connectTimeoutKey{}, ct)
}

func Auth(username string, password string) broker.Option {
	return setBrokerOption(authKey{}, &authRecord{
		username: username,
		password: password,
	})
}

func ConnectHeaders(h map[string]string) broker.Option {
	return setBrokerOption(connectHeaderKey{}, h)
}

func VirtualHost(h string) broker.Option {
	return setBrokerOption(vHostKey{}, h)
}
