package rabbitmq

import (
	"context"
	"fmt"
	"github.com/tx7do/kratos-transport/broker"
	"os"
	"os/signal"
	"syscall"
	"testing"
)

func TestServer(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		Address("amqp://user:bitnami@127.0.0.1:5672"),
		SubscribeDurableQueue("test_topic", "test_topic", receive),
	)

	if err := srv.Start(ctx); err != nil {
		panic(err)
	}

	defer func() {
		if err := srv.Stop(ctx); err != nil {
			t.Errorf("expected nil got %v", err)
		}
	}()

	<-interrupt
}

func receive(event broker.Event) error {
	fmt.Println("Topic: ", event.Topic(), " Payload: ", string(event.Message().Body))
	return nil
}

func TestClient(t *testing.T) {

}
