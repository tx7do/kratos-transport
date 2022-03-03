package websocket

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	ctx := context.Background()

	srv := NewServer(
		Address(":8800"),
		Handle("/ws", receive),
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

func receive(messageType int, payload []byte) (SendBufferArray, error) {
	fmt.Println(" Message Type: ", messageType, " Payload: ", string(payload))

	var messages SendBufferArray

	msg := "hello"
	messages = append(messages, SendBuffer(msg))

	return messages, nil
}
