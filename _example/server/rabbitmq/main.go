package main

import (
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2"
	"log"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/transport/rabbitmq"
)

func main() {
	ctx := context.Background()

	rabbitmqSrv := rabbitmq.NewServer(
		rabbitmq.Address([]string{"amqp://user:bitnami@127.0.0.1:5672"}),
	)

	_ = rabbitmqSrv.RegisterSubscriber(ctx, "test_queue.*", receive)

	app := kratos.New(
		kratos.Name("rabbitmq"),
		kratos.Server(
			rabbitmqSrv,
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
