package mqtt

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	ctx := context.Background()

	srv := NewServer(
		Address("tcp://emqx:public@broker.emqx.io:1883"),
		Topic("topic/bobo/#", 0),
		Handle(receive),
	)

	if err := srv.Start(ctx); err != nil {
		panic(err)
	}

	time.Sleep(time.Second * 60)

	if srv.Stop(ctx) != nil {
		t.Errorf("expected nil got %v", srv.Stop(ctx))
	}
}

func receive(_ context.Context, topic string, payload []byte) error {
	fmt.Println("Topic: ", topic, " Payload: ", string(payload))
	return nil
}
