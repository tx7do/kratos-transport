package main

import (
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2"
	"log"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/transport/kafka"
)

const (
	testBrokers = "localhost:9092"
	testTopic   = "test_topic"
	testGroupId = "a-group"
)

func main() {
	ctx := context.Background()

	kafkaSrv := kafka.NewServer(
		kafka.Address([]string{testBrokers}),
		kafka.Subscribe(ctx, testTopic, testGroupId, false, receive),
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
	fmt.Printf("Topic: %s Payload: %s\n", event.Topic(), string(event.Message().Body))
	return nil
}
