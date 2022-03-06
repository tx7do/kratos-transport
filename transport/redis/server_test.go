package redis

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
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		broker.Addrs("127.0.0.1:6379"),
	)

	if err := srv.Connect(); err != nil {
		panic(err)
	}

	if err := srv.Start(ctx); err != nil {
		panic(err)
	}

	_ = srv.RegisterSubscriber("test_topic", receive,
		broker.SubscribeContext(ctx),
	)

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
