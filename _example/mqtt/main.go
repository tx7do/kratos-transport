package main

import (
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2"
	"github.com/tx7do/kratos-transport/common"
	"github.com/tx7do/kratos-transport/mqtt"
	"log"
)

func main() {
	ctx := context.Background()

	mqttSrv := mqtt.NewServer(
		common.Addrs("tcp://emqx:public@broker.emqx.io:1883"),
		common.OptionContext(ctx),
	)

	_ = mqttSrv.RegisterSubscriber("topic/bobo/#", receive,
		common.SubscribeContext(ctx),
		common.Queue("topic/bobo/#"),
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

func receive(event common.Event) error {
	fmt.Println("Topic: ", event.Topic(), " Payload: ", string(event.Message().Body))
	return nil
}
