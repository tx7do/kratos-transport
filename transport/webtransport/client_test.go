package webtransport

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"testing"

	api "github.com/tx7do/kratos-transport/testing/api/manual"
)

var testClient *Client

func handleClientChatMessage(message *api.ChatMessage) error {
	fmt.Printf("Payload: %v\n", message)
	_ = testClient.SendMessage(api.MessageTypeChat, message)
	return nil
}

func TestClient(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	cli := NewClient(
		//WithClientTLSConfig(NewTlsConfig("", "", "./cert/ca.crt")),
		WithEndpoint("https://localhost:8800/webtransport"),
	)
	defer cli.Disconnect()

	testClient = cli

	cli.RegisterMessageHandler(api.MessageTypeChat,
		func(payload MessagePayload) error {
			switch t := payload.(type) {
			case *api.ChatMessage:
				return handleClientChatMessage(t)
			default:
				return errors.New("invalid payload type")
			}
		},
		func() any { return &api.ChatMessage{} },
	)

	err := cli.Connect()
	if err != nil {
		t.Error(err)
	}

	chatMsg := &api.ChatMessage{
		Message: "Hello, World!",
	}
	_ = cli.SendMessage(api.MessageTypeChat, chatMsg)

	<-interrupt
}
