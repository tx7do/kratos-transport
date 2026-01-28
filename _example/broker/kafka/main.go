package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/broker/kafka"
	api "github.com/tx7do/kratos-transport/testing/api/manual"
)

const (
	testBrokers = "localhost:9092"
	testTopic   = "test_topic"
	testGroupId = "a-group"
)

func handleHygrothermograph(_ context.Context, topic string, headers broker.Headers, msg *api.Hygrothermograph) error {
	log.Infof("Topic %s, Headers: %+v, Payload: %+v\n", topic, headers, msg)
	return nil
}

type HygrothermographHandler func(_ context.Context, topic string, headers broker.Headers, msg *api.Hygrothermograph) error

func RegisterHygrothermographRawHandler(fnc HygrothermographHandler) broker.Handler {
	return func(ctx context.Context, event broker.Event) error {
		var msg api.Hygrothermograph

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

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	b := kafka.NewBroker(
		broker.WithAddress(testBrokers),
		broker.WithCodec("json"),
	)

	_ = b.Init()

	_, err := b.Subscribe(testTopic,
		RegisterHygrothermographRawHandler(handleHygrothermograph),
		api.HygrothermographCreator,
		broker.WithQueueName(testGroupId),
	)
	if err != nil {
		fmt.Println(err)
	}

	<-interrupt
}
