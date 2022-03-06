package rabbitmq

import (
	"context"
	"fmt"
	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/broker/rabbitmq"
	"os"
	"os/signal"
	"syscall"
	"testing"
)

func TestServer(t *testing.T) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

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

	defer func() {
		if err := srv.Stop(ctx); err != nil {
			t.Errorf("expected nil got %v", err)
		}
	}()

	<-sigs
}

func receive(event broker.Event) error {
	fmt.Println("Topic: ", event.Topic(), " Payload: ", string(event.Message().Body))
	return nil
}
