package main

import (
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2"
	"github.com/tx7do/kratos-transport/common"
	"github.com/tx7do/kratos-transport/kafka"
	"log"
)

func main() {
	ctx := context.Background()

	kafkaSrv := kafka.NewServer(
		common.OptionContext(ctx),
		common.Addrs("localhost:9092"),
	)

	if err := kafkaSrv.Connect(); err != nil {
		panic(err)
	}

	_ = kafkaSrv.RegisterSubscriber("test_topic", receive,
		common.SubscribeContext(ctx),
		common.Queue("a-group"),
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

func receive(event common.Event) error {
	fmt.Println("Topic: ", event.Topic(), " Payload: ", string(event.Message().Body))
	return nil
}
