package kafka

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
		broker.Addrs("localhost:9092"),
		broker.OptionContext(ctx),
	)

	_ = srv.RegisterSubscriber("logger.sensor.ts", receive,
		broker.SubscribeContext(ctx),
		broker.Queue("fx-group"),
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
	//_ = event.Ack()
	return nil
}
