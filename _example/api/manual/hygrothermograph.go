package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-transport/broker"
)

type Hygrothermograph struct {
	Humidity    float64 `json:"humidity"`
	Temperature float64 `json:"temperature"`
}

func HygrothermographCreator() broker.Any { return &Hygrothermograph{} }

type HygrothermographHandler func(_ context.Context, topic string, headers broker.Headers, msg *Hygrothermograph) error

func RegisterHygrothermographRawHandler(fnc HygrothermographHandler) broker.Handler {
	return func(ctx context.Context, event broker.Event) error {
		var msg Hygrothermograph

		switch t := event.Message().Body.(type) {
		case []byte:
			if err := json.Unmarshal(t, &msg); err != nil {
				log.Error("json Unmarshal failed: ", err.Error())
				return err
			}
		case string:
			if err := json.Unmarshal([]byte(t), &msg); err != nil {
				log.Error("json Unmarshal failed: ", err.Error())
				return err
			}
		default:
			log.Error("unknown type Unmarshal failed: ", t)
			return fmt.Errorf("unsupported type: %T", t)
		}

		if err := fnc(ctx, event.Topic(), event.Message().Headers, &msg); err != nil {
			return err
		}

		return nil
	}
}

func RegisterHygrothermographJsonHandler(fnc HygrothermographHandler) broker.Handler {
	return func(ctx context.Context, event broker.Event) error {
		switch t := event.Message().Body.(type) {
		case *Hygrothermograph:
			if err := fnc(ctx, event.Topic(), event.Message().Headers, t); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported type: %T", t)
		}
		return nil
	}
}

func RegisterHygrothermographHandler(fnc HygrothermographHandler) broker.Handler {
	return func(ctx context.Context, event broker.Event) error {
		var msg *Hygrothermograph = nil

		switch t := event.Message().Body.(type) {
		case []byte:
			msg = &Hygrothermograph{}
			if err := json.Unmarshal(t, msg); err != nil {
				return err
			}
		case string:
			msg = &Hygrothermograph{}
			if err := json.Unmarshal([]byte(t), msg); err != nil {
				return err
			}
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
