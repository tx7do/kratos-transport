package pulsar_test

import (
	"context"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/broker"
	api "github.com/tx7do/kratos-transport/testing/api/manual"
	"github.com/tx7do/kratos-transport/transport/pulsar"
)

func handleHygrothermograph(_ context.Context, topic string, headers broker.Headers, msg *api.Hygrothermograph) error {
	log.Infof("Topic %s, Headers: %+v, Payload: %+v\n", topic, headers, msg)
	return nil
}

// ExampleNewServer 演示把 Pulsar 订阅挂到 kratos 应用上。
// 没有 Output 注释，go test 只编译不执行；实际运行需要本地 Pulsar (pulsar://127.0.0.1:6650)。
func ExampleNewServer() {
	ctx := context.Background()

	psrSrv := pulsar.NewServer(
		pulsar.WithAddress([]string{"pulsar://127.0.0.1:6650"}),
		pulsar.WithCodec("json"),
	)

	_ = pulsar.RegisterSubscriber(psrSrv, ctx, "test_topic", handleHygrothermograph)

	app := kratos.New(
		kratos.Name("pulsar"),
		kratos.Server(
			psrSrv,
		),
	)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}
