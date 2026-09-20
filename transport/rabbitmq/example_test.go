package rabbitmq_test

import (
	"context"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/broker"
	rabbitmqBroker "github.com/tx7do/kratos-transport/broker/rabbitmq"
	api "github.com/tx7do/kratos-transport/testing/api/manual"
	"github.com/tx7do/kratos-transport/transport/rabbitmq"
)

func handleHygrothermograph(_ context.Context, topic string, headers broker.Headers, msg *api.Hygrothermograph) error {
	log.Infof("Topic %s, Headers: %+v, Payload: %+v\n", topic, headers, msg)
	return nil
}

// ExampleNewServer 演示把 RabbitMQ 订阅挂到 kratos 应用上。
// 没有 Output 注释，go test 只编译不执行；实际运行需要本地 RabbitMQ (amqp://127.0.0.1:5672)。
func ExampleNewServer() {
	ctx := context.Background()

	rabbitmqSrv := rabbitmq.NewServer(
		rabbitmq.WithAddress([]string{"amqp://user:bitnami@127.0.0.1:5672"}),
		rabbitmq.WithCodec("json"),
		rabbitmq.WithExchange("test_exchange", true),
	)

	_ = rabbitmq.RegisterSubscriber(rabbitmqSrv, ctx, "test_routing_key",
		handleHygrothermograph,
		broker.WithSubscribeQueueName("test_queue"),
		rabbitmqBroker.WithDurableQueue())

	app := kratos.New(
		kratos.Name("rabbitmq"),
		kratos.Server(
			rabbitmqSrv,
		),
	)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}
