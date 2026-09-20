package sqs_test

import (
	"context"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/broker"
	api "github.com/tx7do/kratos-transport/testing/api/manual"
	"github.com/tx7do/kratos-transport/transport/sqs"
)

func handleHygrothermograph(_ context.Context, topic string, headers broker.Headers, msg *api.Hygrothermograph) error {
	log.Infof("Topic %s, Headers: %+v, Payload: %+v\n", topic, headers, msg)
	return nil
}

// ExampleNewServer 演示把 AWS SQS 订阅挂到 kratos 应用上。
// 没有 Output 注释，go test 只编译不执行；实际运行需要本地 ElasticMQ 模拟器
// (http://127.0.0.1:9324) 或可访问的 AWS SQS。
func ExampleNewServer() {
	sqsSrv := sqs.NewServer(
		sqs.WithRegion("us-east-1"),
		sqs.WithEndpoint("http://127.0.0.1:9324"),
		sqs.WithCodec("json"),
	)

	_ = sqs.RegisterSubscriber(sqsSrv, "test_topic", handleHygrothermograph)

	app := kratos.New(
		kratos.Name("sqs"),
		kratos.Server(
			sqsSrv,
		),
	)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}
