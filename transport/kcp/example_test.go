package kcp_test

import (
	"fmt"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/transport/kcp"
)

const MessageTypeChat kcp.NetMessageType = iota + 1

var testServer *kcp.Server

type ChatMessage struct {
	Sender    string `json:"sender"`
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
}

// ExampleNewServer 演示启动一个 KCP 聊天服务：
// 收到客户端消息后广播给所有会话。
// 没有 Output 注释，go test 只编译不执行；实际运行需要客户端通过 KCP 协议连接 :8800。
func ExampleNewServer() {
	kcpSrv := kcp.NewServer(
		kcp.WithAddress(":8800"),
		kcp.WithSocketConnectHandler(handleConnect),
		kcp.WithCodec("json"),
		kcp.WithBlockCrypt(kcp.DefaultBlockCryptPassword, kcp.DefaultBlockCryptSalt),
	)

	testServer = kcpSrv

	kcp.RegisterServerMessageHandler(kcpSrv, MessageTypeChat, handleChatMessage)

	app := kratos.New(
		kratos.Name("kcp"),
		kratos.Server(
			kcpSrv,
		),
	)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}

func handleConnect(sessionId kcp.SessionID, connect bool) {
	if connect {
		fmt.Printf("[%s] connected\n", sessionId)
	} else {
		fmt.Printf("[%s] disconnect\n", sessionId)
	}
}

func handleChatMessage(sessionId kcp.SessionID, message *ChatMessage) error {
	fmt.Printf("[%s] Payload: %v\n", sessionId, message)

	testServer.Broadcast(MessageTypeChat, *message)

	return nil
}
