package tcp

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"testing"
)

var testClient *Client

func handleClientChatMessage(message *ChatMessage) error {
	fmt.Printf("Payload: %v\n", message)
	_ = testClient.SendMessage(MessageTypeChat, message)
	return nil
}

func TestClient(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	cli := NewClient(
		WithEndpoint("localhost:8100"),
		WithClientCodec("json"),
	)
	defer cli.Disconnect()

	testClient = cli

	RegisterClientMessageHandler(cli, MessageTypeChat, handleClientChatMessage)

	err := cli.Connect()
	if err != nil {
		t.Error(err)
	}

	chatMsg := &ChatMessage{
		Message: "Hello, World!",
	}
	_ = cli.SendMessage(MessageTypeChat, chatMsg)

	<-interrupt
}
