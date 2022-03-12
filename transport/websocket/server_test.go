package websocket

import (
	"context"
	"fmt"
	ws "github.com/gorilla/websocket"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		Address(":8800"),
		ReadHandle("/ws", handleMessage),
		RegisterHandle(handleRegister),
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

func handleRegister(connectionId string, register bool) {
	if register {
		fmt.Printf("%s registered\n", connectionId)
	} else {
		fmt.Printf("%s unregistered\n", connectionId)
	}
}

func handleMessage(connectionId string, message *Message) (*Message, error) {
	fmt.Printf("[%s] Payload: %s\n", connectionId, string(message.Body))

	var relyMsg Message
	relyMsg.Body = []byte("hello")

	return &relyMsg, nil
}

func TestClient(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	addr := "localhost:8800"

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
			log.Printf("recv: %s", message)
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case t := <-ticker.C:
			err := c.WriteMessage(ws.BinaryMessage, []byte(t.String()))
			if err != nil {
				log.Println("write:", err)
				return
			}
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
