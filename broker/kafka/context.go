package kafka

import (
	"context"
	"github.com/tx7do/kratos-transport/common"
)

func setSubscribeOption(k, v interface{}) common.SubscribeOption {
	return func(o *common.SubscribeOptions) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, k, v)
	}
}

func setBrokerOption(k, v interface{}) common.Option {
	return func(o *common.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, k, v)
	}
}

func setServerSubscriberOption(k, v interface{}) common.SubscriberOption {
	return func(o *common.SubscriberOptions) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, k, v)
	}
}

func setPublishOption(k, v interface{}) common.PublishOption {
	return func(o *common.PublishOptions) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, k, v)
	}
}
