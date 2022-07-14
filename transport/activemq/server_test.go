package activemq

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/tx7do/kratos-transport/broker/stomp"
	"os"
	"os/signal"
	"syscall"
	"testing"

	"github.com/tx7do/kratos-transport/broker"
)

const (
	testBroker = "stomp://127.0.0.1:61613"
	testTopic  = "test_topic"
)

func TestServer(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		Address([]string{testBroker}),
	)

	_ = srv.RegisterSubscriber(ctx, testTopic, receive)

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

func receive(_ context.Context, event broker.Event) error {
	fmt.Printf("Topic: %s Payload: %s\n", event.Topic(), string(event.Message().Body))
	//_ = event.Ack()
	return nil
}

func TestClient(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	b := stomp.NewBroker(
		broker.Addrs(testBroker),
		broker.OptionContext(ctx),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		t.Logf("cant conect to broker, skip: %v", err)
		t.Skip()
	}

	var msg broker.Message
	msg.Body = []byte(`{"Humidity":60, "Temperature":25}`)
	for i := 0; i < 10; i++ {
		err := b.Publish(testTopic, &msg)
		assert.Nil(t, err)
	}

	<-interrupt
}
