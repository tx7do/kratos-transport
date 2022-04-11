package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/broker/redis"
)

func main() {
	ctx := context.Background()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	b := redis.NewBroker(
		broker.Addrs("localhost:6379"),
		broker.OptionContext(ctx),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		fmt.Println(err)
	}
	defer func(b broker.Broker) {
		err := b.Disconnect()
		if err != nil {
			fmt.Println(err)
		}
	}(b)

	_, _ = b.Subscribe("test_topic", receive,
		broker.SubscribeContext(ctx),
	)

	<-interrupt
}

func receive(_ context.Context, event broker.Event) error {
	fmt.Println("Topic: ", event.Topic(), " Payload: ", string(event.Message().Body))
	//_ = event.Ack()
	return nil
}
