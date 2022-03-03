package main

import (
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2"
	"github.com/tx7do/kratos-transport/kafka"
	"log"
)

func main() {
	kafkaSrv := kafka.NewServer(
		kafka.Address("localhost:9092"),
		kafka.GroupID("a-group"),
		kafka.Topics([]string{"test_topic"}),
		kafka.Handle(receive),
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

func receive(_ context.Context, topic string, key string, value []byte) error {
	fmt.Println("topic: ", topic, " key: ", key, " value: ", string(value))
	return nil
}
