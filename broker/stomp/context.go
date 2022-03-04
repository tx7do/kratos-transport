package stomp

import (
	"context"
	"github.com/tx7do/kratos-transport/broker"
)

type authKey struct{}
type connectHeaderKey struct{}
type connectTimeoutKey struct{}
type durableQueueKey struct{}
type receiptKey struct{}
type subscribeHeaderKey struct{}
type suppressContentLengthKey struct{}
type vHostKey struct{}

type authRecord struct {
	username string
	password string
}

func setSubscribeOption(k, v interface{}) broker.SubscribeOption {
	return func(o *broker.SubscribeOptions) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, k, v)
	}
}

func setBrokerOption(k, v interface{}) broker.Option {
	return func(o *broker.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, k, v)
	}
}

func setPublishOption(k, v interface{}) broker.PublishOption {
	return func(o *broker.PublishOptions) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, k, v)
	}
}
