package nats

import (
	"context"
	"github.com/tx7do/kratos-transport/common"
)

func setBrokerOption(k, v interface{}) common.Option {
	return func(o *common.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, k, v)
	}
}
