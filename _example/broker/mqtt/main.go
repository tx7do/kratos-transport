package main

import (
	"context"
	"encoding/json"
	"fmt"
	jsonCodec "github.com/tx7do/kratos-transport/codec/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/broker/mqtt"
)

const (
	EmqxBroker        = "tcp://broker.emqx.io:1883"
	EmqxCnBroker      = "tcp://broker-cn.emqx.io:1883"
	EclipseBroker     = "tcp://mqtt.eclipseprojects.io:1883"
	MosquittoBroker   = "tcp://test.mosquitto.org:1883"
	HiveMQBroker      = "tcp://broker.hivemq.com:1883"
	LocalEmqxBroker   = "tcp://127.0.0.1:1883"
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
	log.Printf("Headers: %+v, Humidity: %.2f Temperature: %.2f\n", headers, msg.Humidity, msg.Temperature)
	return nil
}

func main() {
	ctx := context.Background()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	b := mqtt.NewBroker(
		broker.OptionContext(ctx),
		broker.Codec(jsonCodec.Marshaler{}),
		broker.Addrs(LocalEmqxBroker),
		mqtt.WithCleanSession(false),
		mqtt.WithAuth("user", "bitnami"),
		mqtt.WithClientId("test-client-2"),
	)

	defer func(b broker.Broker) {
		err := b.Disconnect()
		if err != nil {

		}
	}(b)

	if err := b.Connect(); err != nil {
		fmt.Println(err)
	}

	topic := "topic/bobo/#"
	//topicSharedGroup := "$share/g1/topic/bobo/#"
	//topicSharedQueue := "$queue/topic/bobo/#"

	_, err := b.Subscribe(topic,
		registerHygrothermographJsonHandler(),
		func() broker.Any {
			return &Hygrothermograph{}
		},
		broker.SubscribeContext(ctx),
	)
	if err != nil {
		fmt.Println(err)
	}

	<-interrupt
}
