package websocket

import (
	"context"
	"fmt"
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
		Address(":8800"),
		Handle("/ws", receive),
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

func receive(messageType int, payload []byte) (SendBufferArray, error) {
	fmt.Println(" Message Type: ", messageType, " Payload: ", string(payload))

	var messages SendBufferArray

	msg := "hello"
	messages = append(messages, SendBuffer(msg))

	return messages, nil
}
