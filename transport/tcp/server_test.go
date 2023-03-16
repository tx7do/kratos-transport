package tcp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"testing"

	api "github.com/tx7do/kratos-transport/_example/api/manual"
)

var testServer *Server

func TestServer(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		WithAddress(":8100"),
		WithConnectHandle(handleConnect),
		WithCodec("json"),
	)

	srv.RegisterMessageHandler(api.MessageTypeChat,
		func(sessionId SessionID, payload MessagePayload) error {
			switch t := payload.(type) {
			case *api.ChatMessage:
				return handleChatMessage(sessionId, t)
			default:
				return errors.New("invalid payload type")
			}
		},
		func() Any { return &api.ChatMessage{} },
	)

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

func handleConnect(sessionId SessionID, register bool) {
	if register {
		fmt.Printf("%s registered\n", sessionId)
	} else {
		fmt.Printf("%s unregistered\n", sessionId)
	}
}

func handleChatMessage(sessionId SessionID, message *api.ChatMessage) error {
	fmt.Printf("[%s] Payload: %v\n", sessionId, message)

	testServer.Broadcast(api.MessageTypeChat, *message)

	return nil
}
