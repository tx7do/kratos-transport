package webtransport_test

import (
	"errors"
	"fmt"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	api "github.com/tx7do/kratos-transport/testing/api/manual"
	"github.com/tx7do/kratos-transport/transport/webtransport"
)

// ExampleNewServer 演示启动一个 WebTransport（HTTP/3 over QUIC）聊天服务：
// 客户端通过 CONNECT 建立会话，服务端按消息类型分发处理。
// 没有 Output 注释，go test 只编译不执行；实际运行需要 TLS 证书并以 WebTransport 客户端连接 :8800/webtransport。
func ExampleNewServer() {
	wtSrv := webtransport.NewServer(
		webtransport.WithAddress(":8800"),
		webtransport.WithPath("/webtransport"),
		webtransport.WithCodec("json"),
		webtransport.WithTLSConfig(webtransport.NewTlsConfig("./cert/server.key", "./cert/server.crt", "")),
		webtransport.WithConnectHandle(handleConnect),
	)

	wtSrv.RegisterMessageHandler(api.MessageTypeChat,
		func(sessionId webtransport.SessionID, payload webtransport.MessagePayload) error {
			switch t := payload.(type) {
			case *api.ChatMessage:
				return handleChatMessage(sessionId, t)
			default:
				return errors.New("invalid payload struct type")
			}
		},
		func() any { return &api.ChatMessage{} },
	)

	app := kratos.New(
		kratos.Name("webtransport"),
		kratos.Server(
			wtSrv,
		),
	)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}

func handleConnect(sessionId webtransport.SessionID, connect bool) {
	if connect {
		fmt.Printf("[%d] connected\n", sessionId)
	} else {
		fmt.Printf("[%d] disconnect\n", sessionId)
	}
}

func handleChatMessage(sessionId webtransport.SessionID, message *api.ChatMessage) error {
	fmt.Printf("[%d] Payload: %v\n", sessionId, message)

	return nil
}
