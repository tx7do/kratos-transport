package websocket

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	api "github.com/tx7do/kratos-transport/_example/api/manual"
)

var testServer *Server

func TestServer(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		WithAddress(":8100"),
		WithPath("/"),
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

func TestGob(t *testing.T) {
	var msg Message
	msg.Type = api.MessageTypeChat
	msg.Body = []byte("")

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	_ = enc.Encode(msg)

	fmt.Printf("%s\n", string(buf.Bytes()))
}

func TestMessageMarshal(t *testing.T) {
	var msg Message
	msg.Type = 10000
	msg.Body = []byte("Hello World")

	buf, err := msg.Marshal()
	assert.Nil(t, err)

	fmt.Printf("%s\n", string(buf))

	var msg1 Message
	_ = msg1.Unmarshal(buf)

	fmt.Printf("[%d] [%s]\n", msg1.Type, string(msg1.Body))
}
