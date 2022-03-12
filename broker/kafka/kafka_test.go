package kafka

import (
	"context"
	"fmt"
	"github.com/tx7do/kratos-transport/broker"
	"os"
	"os/signal"
	"syscall"
	"testing"
)

func TestSubscribe(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	b := NewBroker(
		broker.Addrs("127.0.0.1:9092"),
		broker.OptionContext(ctx),
	)

	_, _ = b.Subscribe("logger.sensor.ts", receive,
		broker.SubscribeContext(ctx),
		broker.Queue("fx-group"),
	)

	<-interrupt
}

func receive(event broker.Event) error {
	fmt.Println("Topic: ", event.Topic(), " Payload: ", string(event.Message().Body))
	//_ = event.Ack()
	return nil
}

func TestPublish(t *testing.T) {
	ctx := context.Background()

	b := NewBroker(
		broker.Addrs("127.0.0.1:9092"),
		broker.OptionContext(ctx),
	)

	var msg broker.Message
	msg.Body = []byte(`{"Humidity":60, "Temperature":25}`)
	_ = b.Publish("logger.sensor.ts", &msg)
}
