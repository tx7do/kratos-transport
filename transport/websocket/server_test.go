package websocket

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
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

func handleConnect(sessionId SessionID, queries url.Values, register bool) {
	if register {
		fmt.Printf("[%s] registered [%+v]\n", sessionId, queries)
	} else {
		fmt.Printf("[%s] unregistered\n", sessionId)
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
		WithAddress(":10000"),
		WithPath("/"),
		WithConnectHandle(handleConnect),
		WithCodec("json"),
		//WithPayloadType(PayloadTypeText),
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

func TestGob(t *testing.T) {
	var msg BinaryMessage
	msg.Type = MessageTypeChat
	msg.Body = []byte("")

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	_ = enc.Encode(msg)

	fmt.Printf("%s\n", string(buf.Bytes()))
}

func TestMessageMarshal(t *testing.T) {
	var msg BinaryMessage
	msg.Type = 10000
	msg.Body = []byte("Hello World")

	buf, err := msg.Marshal()
	assert.Nil(t, err)

	fmt.Printf("%s\n", string(buf))

	var msg1 BinaryMessage
	_ = msg1.Unmarshal(buf)

	fmt.Printf("[%d] [%s]\n", msg1.Type, string(msg1.Body))
}
