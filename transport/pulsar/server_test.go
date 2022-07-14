package pulsar

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"os/signal"
	"syscall"
	"testing"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/broker/pulsar"
)

const (
	testBroker = "pulsar://127.0.0.1:6650"
	testTopic  = "test_topic"
)

func TestServer(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		WithAddress([]string{testBroker}),
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
	return nil
}

func TestClient(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	b := pulsar.NewBroker(
		broker.OptionContext(ctx),
		broker.Addrs(testBroker),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		t.Logf("cant connect to broker, skip: %v", err)
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
