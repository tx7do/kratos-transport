package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/broker/kafka"
)

const (
	testBrokers = "localhost:9092"
	testTopic   = "test_topic"
	testGroupId = "a-group"
)

func main() {
	ctx := context.Background()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	b := kafka.NewBroker(
		broker.OptionContext(ctx),
		broker.Addrs(testBrokers),
	)

	_, err := b.Subscribe(testTopic, receive,
		broker.SubscribeContext(ctx),
		broker.Queue(testGroupId),
	)
	if err != nil {
		fmt.Println(err)
	}

	<-interrupt
}

func receive(_ context.Context, event broker.Event) error {
	fmt.Printf("Topic: %s Payload: %s\n", event.Topic(), string(event.Message().Body))
	//_ = event.Ack()
	return nil
}
