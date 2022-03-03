package kafka

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	ctx := context.Background()

	srv := NewServer(
		Address("localhost:9092"),
		GroupID("a-group"),
		Topics([]string{"test_topic"}),
		Handle(receive),
	)

	//go func() {
	// start server
	if err := srv.Start(ctx); err != nil {
		panic(err)
	}
	//}()

	time.Sleep(time.Second)

	if srv.Stop(ctx) != nil {
		t.Errorf("expected nil got %v", srv.Stop(ctx))
	}
}

func receive(_ context.Context, topic string, key string, value []byte) error {
	fmt.Println("topic: ", topic, " key: ", key, " value: ", string(value))
	return nil
}
