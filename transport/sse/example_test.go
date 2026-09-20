package sse_test

import (
	"context"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/transport/sse"
)

var testServer *sse.Server

type NoticeMessage struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// ExampleNewServer 演示启动一个 SSE 事件推送服务：
// 客户端通过 HTTP 长连接订阅 /events，有新订阅者时向所有流广播 JSON 消息。
// 没有 Output 注释，go test 只编译不执行；实际运行需要用 HTTP 客户端订阅 :8800/events。
func ExampleNewServer() {
	sseSrv := sse.NewServer(
		sse.WithAddress(":8800"),
		sse.WithPath("/events"),
		sse.WithCodec("json"),
		sse.WithSubscriberFunction(handleSubscribe),
	)

	testServer = sseSrv

	sseSrv.CreateStream("events")

	app := kratos.New(
		kratos.Name("sse"),
		kratos.Server(
			sseSrv,
		),
	)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}

func handleSubscribe(streamId sse.StreamID, _ *sse.Subscriber) {
	log.Infof("stream [%s] subscribed", streamId)

	if err := testServer.NotifyData(context.Background(),
		&NoticeMessage{Title: "welcome", Body: "hello sse"},
	); err != nil {
		log.Error(err)
	}
}
