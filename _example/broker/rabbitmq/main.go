package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/broker/rabbitmq"
)

func main() {
	ctx := context.Background()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	b := rabbitmq.NewBroker(
		broker.Addrs("amqp://user:bitnami@127.0.0.1:5672"),
		broker.OptionContext(ctx),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		fmt.Println(err)
	}

	_, _ = b.Subscribe("test_queue.*", receive,
		broker.SubscribeContext(ctx),
		broker.Queue("test_queue.*"),
		// broker.DisableAutoAck(),
		rabbitmq.DurableQueue(),
	)

	<-sigs
}

func receive(event broker.Event) error {
	fmt.Println("Topic: ", event.Topic(), " Payload: ", string(event.Message().Body))
	//_ = event.Ack()
	return nil
}
