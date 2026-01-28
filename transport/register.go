package transport

import (
	"context"
	"fmt"

	"github.com/tx7do/kratos-transport/broker"
)

type SubscriberRegistrar interface {
	RegisterSubscriber(
		ctx context.Context,
		topic, queue string,
		disableAutoAck bool,
		handler broker.Handler,
		binder broker.Binder,
		opts ...broker.SubscribeOption,
	) error
}

func RegisterSubscriber[S SubscriberRegistrar, T any](
	srv S,
	ctx context.Context,
	topic, queue string,
	disableAutoAck bool,
	handler broker.TypedHandler[T],
	opts ...broker.SubscribeOption,
) error {
	return srv.RegisterSubscriber(ctx,
		topic,
		queue,
		disableAutoAck,
		func(ctx context.Context, event broker.Event) error {
			if event == nil {
				return fmt.Errorf("event is nil")
			}

			switch t := event.Message().Body.(type) {
			case *T:
				if err := handler(ctx, event.Topic(), event.Message().Headers, t); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported type: %T", t)
			}
			return nil
		},
		func() any {
			var t T
			return &t
		},
		opts...,
	)
}
