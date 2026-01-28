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
	"github.com/tx7do/kratos-transport/broker/mqtt"
	api "github.com/tx7do/kratos-transport/testing/api/manual"
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

	b := mqtt.NewBroker(
		broker.WithCodec("json"),
		broker.WithAddress(LocalEmqxBroker),
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
	defer b.Disconnect()

	topic := "topic/bobo/#"
	//topicSharedGroup := "$share/g1/topic/bobo/#"
	//topicSharedQueue := "$queue/topic/bobo/#"

	_, err := b.Subscribe(topic,
		RegisterHygrothermographRawHandler(handleHygrothermograph),
		api.HygrothermographCreator,
	)
	if err != nil {
		fmt.Println(err)
	}

	<-interrupt
}
