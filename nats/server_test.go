package nats

import (
	"context"
	"fmt"
	"github.com/tx7do/kratos-transport/broker"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	ctx := context.Background()

	srv := NewServer(
		broker.Addrs("nats://127.0.0.1:4222"),
		broker.OptionContext(ctx),
	)

	_ = srv.RegisterSubscriber("test_topic", receive,
		broker.SubscribeContext(ctx),
		broker.Queue("test_topic"),
	)

	if err := srv.Start(ctx); err != nil {
		panic(err)
	}

	time.Sleep(time.Second * 60)

	if srv.Stop(ctx) != nil {
		t.Errorf("expected nil got %v", srv.Stop(ctx))
	}
}

func receive(event broker.Event) error {
	fmt.Println("Topic: ", event.Topic(), " Payload: ", string(event.Message().Body))
	return nil
}
