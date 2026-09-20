package signalr_test

import (
	"fmt"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	signalrLib "github.com/philippseith/signalr"

	"github.com/tx7do/kratos-transport/transport/signalr"
)

// ChatHub 聊天 Hub：客户端调用 Broadcast 方法后，消息回推给 group 内所有连接。
type ChatHub struct {
	signalrLib.Hub
}

func (h *ChatHub) OnConnected(connectionID string) {
	fmt.Printf("[%s] connected\n", connectionID)
	h.Groups().AddToGroup("chat", connectionID)
}

func (h *ChatHub) OnDisconnected(connectionID string) {
	fmt.Printf("[%s] disconnected\n", connectionID)
	h.Groups().RemoveFromGroup("chat", connectionID)
}

func (h *ChatHub) Broadcast(message string) {
	h.Clients().Group("chat").Send("receive", message)
}

// ExampleNewServer 演示启动一个 SignalR 聊天 Hub 服务：
// Hub 挂载到 /chat 路径，客户端加入 group 后互发消息。
// 没有 Output 注释，go test 只编译不执行；实际运行需要 SignalR 客户端连接 :8800/chat。
func ExampleNewServer() {
	signalrSrv := signalr.NewServer(
		signalr.WithAddress(":8800"),
		signalr.WithCodec("json"),
		signalr.WithHub(&ChatHub{}),
	)

	signalrSrv.MapHTTP("/chat")

	app := kratos.New(
		kratos.Name("signalr"),
		kratos.Server(
			signalrSrv,
		),
	)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}
