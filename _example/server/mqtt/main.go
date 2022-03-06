package main

import (
	"context"
	"fmt"
	"log"

	"github.com/go-kratos/kratos/v2"
	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/transport/mqtt"
)

func main() {
	ctx := context.Background()

	mqttSrv := mqtt.NewServer(
		broker.Addrs("tcp://emqx:public@broker.emqx.io:1883"),
		broker.OptionContext(ctx),
	)

	_ = mqttSrv.RegisterSubscriber("topic/bobo/#", receive,
		broker.SubscribeContext(ctx),
		broker.Queue("topic/bobo/#"),
	)

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

func receive(event broker.Event) error {
	fmt.Println("Topic: ", event.Topic(), " Payload: ", string(event.Message().Body))
	return nil
}
