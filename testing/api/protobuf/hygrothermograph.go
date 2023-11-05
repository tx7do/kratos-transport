package api

import (
	"context"
	"fmt"
	"github.com/tx7do/kratos-transport/broker"
)

func HygrothermographCreator() broker.Any { return &Hygrothermograph{} }

type HygrothermographHandler func(_ context.Context, topic string, headers broker.Headers, msg *Hygrothermograph) error

func RegisterHygrothermographHandler(fnc HygrothermographHandler) broker.Handler {
	return func(ctx context.Context, event broker.Event) error {
		var msg *Hygrothermograph = nil

		switch t := event.Message().Body.(type) {
		case *Hygrothermograph:
			msg = t
		default:
			return fmt.Errorf("unsupported type: %T", t)
		}

		if err := fnc(ctx, event.Topic(), event.Message().Headers, msg); err != nil {
			return err
		}

		return nil
	}
}
