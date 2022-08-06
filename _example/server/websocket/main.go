package main

import (
	"fmt"
	"log"

	"github.com/go-kratos/kratos/v2"
	"github.com/tx7do/kratos-transport/transport/websocket"
)

var testServer *websocket.Server

func main() {
	//ctx := context.Background()

	wsSrv := websocket.NewServer(
		websocket.WithAddress(":8800"),
		websocket.WithReadHandle("/ws", handleMessage),
		websocket.WithConnectHandle(handleConnect),
	)

	testServer = wsSrv

	app := kratos.New(
		kratos.Name("websocket"),
		kratos.Server(
			wsSrv,
		),
	)
	if err := app.Run(); err != nil {
		log.Println(err)
	}
}

func handleConnect(connectionId string, register bool) {
	if register {
		fmt.Printf("%s connected\n", connectionId)
	} else {
		fmt.Printf("%s disconnect\n", connectionId)
	}
}

func handleMessage(connectionId string, message *websocket.Message) error {
	fmt.Printf("[%s] Payload: %s\n", connectionId, string(message.Body))

	var relyMsg websocket.Message
	relyMsg.Body = []byte("hello")

	testServer.SendMessage(connectionId, &relyMsg)

	return nil
}
