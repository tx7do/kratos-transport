package rabbitmq

import (
	"context"
	"fmt"
	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/broker/rabbitmq"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	ctx := context.Background()

	srv := NewServer(
		broker.Addrs("amqp://user:bitnami@127.0.0.1:5672"),
		broker.OptionContext(ctx),
	)

	_ = srv.RegisterSubscriber("test_topic", receive,
		broker.SubscribeContext(ctx),
		broker.Queue("test_topic"),
		//common.DisableAutoAck(),
		rabbitmq.DurableQueue(),
	)

	if err := srv.Start(ctx); err != nil {
		panic(err)
	}

	time.Sleep(time.Minute * 60)

	if srv.Stop(ctx) != nil {
		t.Errorf("expected nil got %v", srv.Stop(ctx))
	}
}

func receive(event broker.Event) error {
	fmt.Println("Topic: ", event.Topic(), " Payload: ", string(event.Message().Body))
	return nil
}
