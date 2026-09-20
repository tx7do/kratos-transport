package rocketmq_test

import (
	"context"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/broker"
	rocketmqOption "github.com/tx7do/kratos-transport/broker/rocketmq/option"
	api "github.com/tx7do/kratos-transport/testing/api/manual"
	"github.com/tx7do/kratos-transport/transport/rocketmq"
)

func handleHygrothermograph(_ context.Context, topic string, headers broker.Headers, msg *api.Hygrothermograph) error {
	log.Infof("Topic %s, Headers: %+v, Payload: %+v\n", topic, headers, msg)
	return nil
}

// ExampleNewServer 演示把 RocketMQ 订阅挂到 kratos 应用上。
// 没有 Output 注释，go test 只编译不执行；实际运行需要本地 RocketMQ NameServer (127.0.0.1:9876)。
func ExampleNewServer() {
	ctx := context.Background()

	rmqSrv := rocketmq.NewServer(
		rocketmqOption.DriverTypeV2,
		rocketmq.WithNameServer([]string{"127.0.0.1:9876"}),
		rocketmq.WithCodec("json"),
	)

	_ = rocketmq.RegisterSubscriber(rmqSrv, ctx, "test_topic", "CID_TEST_GROUP", handleHygrothermograph)

	app := kratos.New(
		kratos.Name("rocketmq"),
		kratos.Server(
			rmqSrv,
		),
	)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}
