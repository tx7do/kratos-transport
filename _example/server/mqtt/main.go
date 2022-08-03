package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/go-kratos/kratos/v2"
	"github.com/tx7do/kratos-transport/broker"
	jsonCodec "github.com/tx7do/kratos-transport/codec/json"
	"github.com/tx7do/kratos-transport/transport/mqtt"
)

const (
	EmqxBroker        = "tcp://broker.emqx.io:1883"
	EmqxCnBroker      = "tcp://broker-cn.emqx.io:1883"
	EclipseBroker     = "tcp://mqtt.eclipseprojects.io:1883"
	MosquittoBroker   = "tcp://test.mosquitto.org:1883"
	HiveMQBroker      = "tcp://broker.hivemq.com:1883"
	LocalEmxqBroker   = "tcp://127.0.0.1:1883"
	LocalRabbitBroker = "tcp://user:bitnami@127.0.0.1:1883"
)

type Hygrothermograph struct {
	Humidity    float64 `json:"humidity"`
	Temperature float64 `json:"temperature"`
}

func registerHygrothermographRawHandler() broker.Handler {
	return func(ctx context.Context, event broker.Event) error {
		var msg Hygrothermograph

		switch t := event.Message().Body.(type) {
		case []byte:
			if err := json.Unmarshal(t, &msg); err != nil {
				return err
			}
		case string:
			if err := json.Unmarshal([]byte(t), &msg); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported type: %T", t)
		}

		if err := handleHygrothermograph(ctx, event.Topic(), event.Message().Headers, &msg); err != nil {
			return err
		}

		return nil
	}
}

func registerHygrothermographJsonHandler() broker.Handler {
	return func(ctx context.Context, event broker.Event) error {
		switch t := event.Message().Body.(type) {
		case *Hygrothermograph:
			if err := handleHygrothermograph(ctx, event.Topic(), event.Message().Headers, t); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported type: %T", t)
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

	mqttSrv := mqtt.NewServer(
		mqtt.WithAddress([]string{EmqxCnBroker}),
		mqtt.WithCodec(jsonCodec.Marshaler{}),
	)

	_ = mqttSrv.RegisterSubscriber(ctx,
		"topic/bobo/#",
		registerHygrothermographJsonHandler(),
		func() broker.Any {
			return &Hygrothermograph{}
		})

	app := kratos.New(
		kratos.Name("mqtt"),
		kratos.Server(
			mqttSrv,
		),
	)
	if err := app.Run(); err != nil {
		log.Println(err)
	}
}
