package rabbitmq

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/tx7do/kratos-transport/broker"
	"os"
	"os/signal"
	"syscall"
	"testing"
)

func TestDurableQueueSubscribe(t *testing.T) {

	ctx := context.Background()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	b := NewBroker(
		broker.OptionContext(ctx),
		broker.Addrs("amqp://user:bitnami@127.0.0.1:5672/"),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		t.Logf("cant conect to broker, skip: %v", err)
		t.Skip()
	}

	_, err := b.Subscribe("logger.sensor.ts", receive,
		broker.SubscribeContext(ctx),
		broker.Queue("logger.sensor.ts"),
		// broker.DisableAutoAck(),
		DurableQueue(),
	)
	assert.Nil(t, err)

	<-interrupt
}

func receive(_ context.Context, event broker.Event) error {
	fmt.Println("Topic: ", event.Topic(), " Payload: ", string(event.Message().Body))
	//_ = event.Ack()
	return nil
}

func TestPublish(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	b := NewBroker(
		broker.OptionContext(ctx),
		broker.Addrs("amqp://user:bitnami@127.0.0.1:5672/"),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		t.Logf("cant conect to broker, skip: %v", err)
		t.Skip()
	}

	var msg broker.Message
	msg.Body = []byte(`{"Humidity":60, "Temperature":25}`)
	_ = b.Publish("amq.topic", &msg)

	<-interrupt
}
