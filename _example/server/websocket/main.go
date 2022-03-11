package main

import (
	"fmt"
	"log"

	"github.com/go-kratos/kratos/v2"
	"github.com/tx7do/kratos-transport/transport/websocket"
)

func main() {
	//ctx := context.Background()

	wsSrv := websocket.NewServer(
		websocket.Address(":8800"),
		websocket.Handle("/ws", receive),
	)

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

func receive(message *websocket.Message) (*websocket.Message, error) {
	fmt.Println(" Payload: ", string(message.Body))

	var relyMsg websocket.Message
	relyMsg.Body = []byte("hello")

	return &relyMsg, nil
}
