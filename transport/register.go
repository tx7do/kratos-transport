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
			if event == nil || event.Message() == nil || event.Message().Body == nil {
				return fmt.Errorf("event or message body is nil")
			}

			var zero T
			expectedType := fmt.Sprintf("%T", &zero)

			switch t := event.Message().Body.(type) {
			case *T:
				if err := handler(ctx, event.Topic(), event.Message().Headers, t); err != nil {
					return err
				}
			case T:
				if err := handler(ctx, event.Topic(), event.Message().Headers, &t); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported type: expected %s, got %T", expectedType, event.Message().Body)
			}

			// 手动 ack 模式（disableAutoAck）：typed handler 拿不到 Event，
			// 约定为处理成功即提交；失败不提交，等重投
			if disableAutoAck {
				return event.Ack()
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
