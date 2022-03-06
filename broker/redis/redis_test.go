package redis

import (
	"context"
	"fmt"
	"github.com/tx7do/kratos-transport/broker"
	"os"
	"os/signal"
	"reflect"
	"sort"
	"syscall"
	"testing"
	"time"
)

func subscribe(t *testing.T, b broker.Broker, topic string, handle broker.Handler) broker.Subscriber {
	s, err := b.Subscribe(topic, handle)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func publish(t *testing.T, b broker.Broker, topic string, msg *broker.Message) {
	if err := b.Publish(topic, msg); err != nil {
		t.Fatal(err)
	}
}

func unsubscribe(t *testing.T, s broker.Subscriber) {
	if err := s.Unsubscribe(); err != nil {
		t.Fatal(err)
	}
}

func TestBroker(t *testing.T) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	b := NewBroker(
		broker.Addrs("localhost:6379"),
	)

	// Only setting options.
	_ = b.Init()

	if err := b.Connect(); err != nil {
		t.Fatal(err)
	}
	defer func(b broker.Broker) {
		err := b.Disconnect()
		if err != nil {

		}
	}(b)

	// Large enough buffer to not block.
	msgs := make(chan string, 10)

	go func() {
		s1 := subscribe(t, b, "test", func(p broker.Event) error {
			m := p.Message()
			msgs <- fmt.Sprintf("s1:%s", string(m.Body))
			return nil
		})

		s2 := subscribe(t, b, "test", func(p broker.Event) error {
			m := p.Message()
			msgs <- fmt.Sprintf("s2:%s", string(m.Body))
			return nil
		})

		publish(t, b, "test", &broker.Message{
			Body: []byte("hello"),
		})

		publish(t, b, "test", &broker.Message{
			Body: []byte("world"),
		})

		unsubscribe(t, s1)

		publish(t, b, "test", &broker.Message{
			Body: []byte("other"),
		})

		unsubscribe(t, s2)

		publish(t, b, "test", &broker.Message{
			Body: []byte("none"),
		})

		close(msgs)
	}()

	var actual []string
	for msg := range msgs {
		actual = append(actual, msg)
	}

	exp := []string{
		"s1:hello",
		"s2:hello",
		"s1:world",
		"s2:world",
		"s2:other",
	}

	// Order is not guaranteed.
	sort.Strings(actual)
	sort.Strings(exp)

	if !reflect.DeepEqual(actual, exp) {
		t.Fatalf("expected %v, got %v", exp, actual)
	}

	<-sigs
}

func TestReceive(t *testing.T) {
	ctx := context.Background()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	b := NewBroker(
		broker.Addrs("redis://localhost:6379"),
		//IdleTimeout(24*time.Hour),
		ReadTimeout(24*time.Hour),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		fmt.Println(err)
	}
	defer func(b broker.Broker) {
		err := b.Disconnect()
		if err != nil {
			fmt.Println(err)
		}
	}(b)

	_, _ = b.Subscribe("test_topic", receive,
		broker.SubscribeContext(ctx),
	)

	<-sigs
}

func receive(event broker.Event) error {
	fmt.Println("Topic: ", event.Topic(), " Payload: ", string(event.Message().Body))
	//_ = event.Ack()
	return nil
}
