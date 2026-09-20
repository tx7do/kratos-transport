package tcp_test

import (
	"fmt"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/transport/tcp"
)

const MessageTypeChat tcp.NetMessageType = iota + 1

var testServer *tcp.Server

type ChatMessage struct {
	Sender    string `json:"sender"`
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
}

// ExampleNewServer 演示启动一个 TCP 聊天服务：
// 收到客户端消息后广播给所有会话。
// 没有 Output 注释，go test 只编译不执行；实际运行需要客户端通过 TCP 长连接 :8800。
func ExampleNewServer() {
	tcpSrv := tcp.NewServer(
		tcp.WithAddress(":8800"),
		tcp.WithSocketConnectHandler(handleConnect),
		tcp.WithCodec("json"),
	)

	testServer = tcpSrv

	tcp.RegisterServerMessageHandler(tcpSrv, MessageTypeChat, handleChatMessage)

	app := kratos.New(
		kratos.Name("tcp"),
		kratos.Server(
			tcpSrv,
		),
	)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}

func handleConnect(sessionId tcp.SessionID, connect bool) {
	if connect {
		fmt.Printf("[%s] connected\n", sessionId)
	} else {
		fmt.Printf("[%s] disconnect\n", sessionId)
	}
}

func handleChatMessage(sessionId tcp.SessionID, message *ChatMessage) error {
	fmt.Printf("[%s] Payload: %v\n", sessionId, message)

	testServer.Broadcast(MessageTypeChat, *message)

	return nil
}
