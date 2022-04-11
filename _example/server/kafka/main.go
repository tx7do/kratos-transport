package main

import (
	"fmt"
	"github.com/go-kratos/kratos/v2"
	"log"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/transport/kafka"
)

func main() {
	//ctx := context.Background()

	kafkaSrv := kafka.NewServer(
		kafka.Address("127.0.0.1:9092"),
		kafka.Subscribe("test_topic", "a-group", receive),
	)

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

func receive(_ context.Context, event broker.Event) error {
	fmt.Println("Topic: ", event.Topic(), " Payload: ", string(event.Message().Body))
	return nil
}
