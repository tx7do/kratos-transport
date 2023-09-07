package tcp

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"testing"
)

var testServer *Server

const (
	MessageTypeChat = iota + 1
)

type ChatMessage struct {
	Type    int    `json:"type"`
	Sender  string `json:"sender"`
	Message string `json:"message"`
}

func handleConnect(sessionId SessionID, register bool) {
	if register {
		fmt.Printf("%s registered\n", sessionId)
	} else {
		fmt.Printf("%s unregistered\n", sessionId)
	}
}

func handleChatMessage(sessionId SessionID, message *ChatMessage) error {
	fmt.Printf("[%s] Payload: %v\n", sessionId, message)

	testServer.Broadcast(MessageTypeChat, *message)

	return nil
}

func TestServer(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		WithAddress(":8100"),
		WithConnectHandler(handleConnect),
		WithCodec("json"),
	)

	RegisterServerMessageHandler(srv, MessageTypeChat, handleChatMessage)

	testServer = srv

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
