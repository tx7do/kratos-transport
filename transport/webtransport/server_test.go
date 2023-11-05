package webtransport

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"testing"

	api "github.com/tx7do/kratos-transport/testing/api/manual"
)

var testServer *Server

func handleConnect(sessionId SessionID, register bool) {
	if register {
		fmt.Printf("%d registered\n", sessionId)
	} else {
		fmt.Printf("%d unregistered\n", sessionId)
	}
}

func handleChatMessage(sessionId SessionID, message *api.ChatMessage) error {
	fmt.Printf("[%d] Payload: %v\n", sessionId, message)

	//testServer.Broadcast(api.MessageTypeChat, *message)

	return nil
}

func TestServer(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	// https://localhost:8800/webtransport
	srv := NewServer(
		WithAddress(":8800"),
		WithPath("/webtransport"),
		WithConnectHandle(handleConnect),
		WithTLSConfig(NewTlsConfig("./cert/server.key", "./cert/server.crt", "")),
		//WithCodec("json"),
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
