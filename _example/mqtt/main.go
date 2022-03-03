package main

import (
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2"
	"github.com/tx7do/kratos-transport/mqtt"
	"log"
)

func main() {
	mqttSrv := mqtt.NewServer(
		mqtt.Address("tcp://emqx:public@broker.emqx.io:1883"),
		mqtt.Topic("topic/bobo/#", 0),
		mqtt.Handle(receive),
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

func receive(_ context.Context, topic string, payload []byte) error {
	fmt.Println("Topic: ", topic, " Payload: ", string(payload))
	return nil
}
