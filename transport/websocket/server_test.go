package websocket

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/go-kratos/kratos/v2/encoding"
	ws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/tx7do/kratos-transport/broker"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

var testServer *Server

const (
	MessageTypeChat = iota + 1
)

type ChatMessage struct {
	Type    int    `json:"type"`
	Message string `json:"message"`
}

func TestServer(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		WithAddress(":8800"),
		WithPath("/ws"),
		WithConnectHandle(handleConnect),
		WithCodec(encoding.GetCodec("json")),
	)

	srv.RegisterMessageHandler(MessageTypeChat,
		func(sessionId SessionID, payload MessagePayload) error {
			switch t := payload.(type) {
			case *ChatMessage:
				return handleChatMessage(sessionId, t)
			default:
				return errors.New("invalid payload type")
			}
		},
		func() Any { return &ChatMessage{} },
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

func handleChatMessage(sessionId SessionID, message *ChatMessage) error {
	fmt.Printf("[%s] Payload: %v\n", sessionId, message)

	testServer.Broadcast(MessageTypeChat, *message)

	return nil
}

func TestClient(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	addr := "localhost:8800"
	codec := encoding.GetCodec("json")

	u := url.URL{Scheme: "ws", Host: addr, Path: "/ws"}
	log.Printf("connecting to %s", u.String())

	c, _, err := ws.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer func(c *ws.Conn) {
		err := c.Close()
		if err != nil {
			log.Printf("disconnect failed: %s", err.Error())
		}
	}(c)

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			var msg Message
			_ = msg.Unmarshal(message)
			var chatMsg ChatMessage
			_ = broker.Unmarshal(codec, msg.Body, &chatMsg)
			fmt.Printf("Received: %v\n", chatMsg)
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			chatMsg := ChatMessage{}
			chatMsg.Type = 100
			chatMsg.Message = "Hello World"
			var msg Message
			msg.Type = MessageTypeChat
			msg.Body, _ = broker.Marshal(codec, chatMsg)

			buff, _ := msg.Marshal()
			_ = c.WriteMessage(ws.BinaryMessage, buff)
		case <-interrupt:
			log.Println("interrupt")

			err := c.WriteMessage(ws.CloseMessage, ws.FormatCloseMessage(ws.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}

func TestGob(t *testing.T) {
	var msg Message
	msg.Type = MessageTypeChat
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
