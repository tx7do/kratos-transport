package redis

import (
	"fmt"
	"github.com/tx7do/kratos-transport/common"
	"os"
	"reflect"
	"sort"
	"testing"
)

func subscribe(t *testing.T, b common.Broker, topic string, handle common.Handler) common.Subscriber {
	s, err := b.Subscribe(topic, handle)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func publish(t *testing.T, b common.Broker, topic string, msg *common.Message) {
	if err := b.Publish(topic, msg); err != nil {
		t.Fatal(err)
	}
}

func unsubscribe(t *testing.T, s common.Subscriber) {
	if err := s.Unsubscribe(); err != nil {
		t.Fatal(err)
	}
}

func TestBroker(t *testing.T) {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		t.Skip("REDIS_URL not defined")
	}

	b := NewBroker(common.Addrs(url))

	// Only setting options.
	b.Init()

	if err := b.Connect(); err != nil {
		t.Fatal(err)
	}
	defer b.Disconnect()

	// Large enough buffer to not block.
	msgs := make(chan string, 10)

	go func() {
		s1 := subscribe(t, b, "test", func(p common.Event) error {
			m := p.Message()
			msgs <- fmt.Sprintf("s1:%s", string(m.Body))
			return nil
		})

		s2 := subscribe(t, b, "test", func(p common.Event) error {
			m := p.Message()
			msgs <- fmt.Sprintf("s2:%s", string(m.Body))
			return nil
		})

		publish(t, b, "test", &common.Message{
			Body: []byte("hello"),
		})

		publish(t, b, "test", &common.Message{
			Body: []byte("world"),
		})

		unsubscribe(t, s1)

		publish(t, b, "test", &common.Message{
			Body: []byte("other"),
		})

		unsubscribe(t, s2)

		publish(t, b, "test", &common.Message{
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
}
