package websocket

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"testing"

	api "github.com/tx7do/kratos-transport/_example/api/manual"
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
		WithEndpoint("ws://localhost:8100/"),
		WithClientCodec("json"),
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
		func() Any { return &api.ChatMessage{} },
	)

	err := cli.Connect()
	if err != nil {
		t.Error(err)
	}

	//chatMsg := &api.ChatMessage{
	//	Message: "Hello, World!",
	//}
	//_ = cli.SendMessage(api.MessageTypeChat, chatMsg)

	<-interrupt
}

var keyGUID = []byte("258EAFA5-E914-47DA-95CA-C5AB0DC85B11")

func computeAcceptKey(challengeKey string) string {
	h := sha1.New()
	h.Write([]byte(challengeKey))
	h.Write(keyGUID)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func generateChallengeKey() (string, error) {
	p := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, p); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(p), nil
}

func Test1(t *testing.T) {
	challengeKey, _ := generateChallengeKey()
	fmt.Println(computeAcceptKey(challengeKey))

	fmt.Println(computeAcceptKey("foIGUMVOg/QOba9qZkaCmg=="))
	fmt.Println(computeAcceptKey("UHF9V2jktxC//1zmwLnxMg=="))
	fmt.Println(computeAcceptKey("KWtssYGuj2uQiv7bG7tc7A=="))
	fmt.Println(computeAcceptKey("3G4O+cC9DDGJS9pJAhzpUA=="))
}
