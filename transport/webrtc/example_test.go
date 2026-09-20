package webrtc_test

import (
	"fmt"
	"net/url"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/transport/webrtc"
)

const MessageTypeChat webrtc.NetMessageType = iota + 1

var testServer *webrtc.Server

type ChatMessage struct {
	Sender    string `json:"sender"`
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
}

// ExampleNewServer 演示启动一个 WebRTC 信令服务：
// 客户端向 /signal POST Offer 完成协商，数据通道建立后收到消息即广播给所有会话。
// 没有 Output 注释，go test 只编译不执行；实际运行需要客户端完成 ICE/SDP 信令交换。
func ExampleNewServer() {
	rtcSrv := webrtc.NewServer(
		webrtc.WithAddress(":8800"),
		webrtc.WithPath("/signal"),
		webrtc.WithCodec("json"),
		webrtc.WithSocketConnectHandler(handleConnect),
	)

	testServer = rtcSrv

	webrtc.RegisterServerMessageHandler(rtcSrv, MessageTypeChat, handleChatMessage)

	app := kratos.New(
		kratos.Name("webrtc"),
		kratos.Server(
			rtcSrv,
		),
	)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}

func handleConnect(sessionId webrtc.SessionID, queries url.Values, connect bool) {
	if connect {
		fmt.Printf("[%s] connected [%+v]\n", sessionId, queries)
	} else {
		fmt.Printf("[%s] disconnect\n", sessionId)
	}
}

func handleChatMessage(sessionId webrtc.SessionID, message *ChatMessage) error {
	fmt.Printf("[%s] Payload: %v\n", sessionId, message)

	testServer.Broadcast(MessageTypeChat, *message)

	return nil
}
