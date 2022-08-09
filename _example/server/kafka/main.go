package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/transport/kafka"
)

const (
	testBrokers = "localhost:9092"
	testTopic   = "test_topic"
	testGroupId = "a-group"
)

type Hygrothermograph struct {
	Humidity    float64 `json:"humidity"`
	Temperature float64 `json:"temperature"`
}

func registerHygrothermographHandler() broker.Handler {
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

		if err := handleHygrothermograph(ctx, event.Topic(), event.Message().Headers, msg); err != nil {
			return err
		}

		return nil
	}
}

func handleHygrothermograph(_ context.Context, topic string, headers broker.Headers, msg *Hygrothermograph) error {
	log.Printf("Humidity: %.2f Temperature: %.2f\n", msg.Humidity, msg.Temperature)
	return nil
}

func main() {
	ctx := context.Background()

	kafkaSrv := kafka.NewServer(
		kafka.WithAddress([]string{testBrokers}),
		kafka.WithCodec(encoding.GetCodec("json")),
	)

	_ = kafkaSrv.RegisterSubscriber(ctx,
		testTopic, testGroupId, false,
		registerHygrothermographHandler(),
		func() broker.Any {
			return &Hygrothermograph{}
		})

	app := kratos.New(
		kratos.Name("kafka"),
		kratos.Server(
			kafkaSrv,
		),
	)
	if err := app.Run(); err != nil {
		log.Println(err)
	}
}
