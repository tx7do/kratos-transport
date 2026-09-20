package activemq_test

import (
	"context"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/broker"
	api "github.com/tx7do/kratos-transport/testing/api/manual"
	"github.com/tx7do/kratos-transport/transport/activemq"
)

func handleHygrothermograph(_ context.Context, topic string, headers broker.Headers, msg *api.Hygrothermograph) error {
	log.Infof("Topic %s, Headers: %+v, Payload: %+v\n", topic, headers, msg)
	return nil
}

// ExampleNewServer 演示把 ActiveMQ (STOMP) 订阅挂到 kratos 应用上。
// 没有 Output 注释，go test 只编译不执行；实际运行需要本地 ActiveMQ (stomp://127.0.0.1:61613)。
func ExampleNewServer() {
	ctx := context.Background()

	amqSrv := activemq.NewServer(
		activemq.WithAddress([]string{"stomp://127.0.0.1:61613"}),
		activemq.WithCodec("json"),
	)

	_ = activemq.RegisterSubscriber(amqSrv, ctx, "test_topic", handleHygrothermograph)

	app := kratos.New(
		kratos.Name("activemq"),
		kratos.Server(
			amqSrv,
		),
	)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}
