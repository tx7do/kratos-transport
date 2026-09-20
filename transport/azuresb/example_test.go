package azuresb_test

import (
	"context"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/broker"
	api "github.com/tx7do/kratos-transport/testing/api/manual"
	"github.com/tx7do/kratos-transport/transport/azuresb"
)

func handleHygrothermograph(_ context.Context, topic string, headers broker.Headers, msg *api.Hygrothermograph) error {
	log.Infof("Topic %s, Headers: %+v, Payload: %+v\n", topic, headers, msg)
	return nil
}

// ExampleNewServer 演示把 Azure Service Bus 订阅挂到 kratos 应用上。
// 没有 Output 注释，go test 只编译不执行；实际运行需要可连的 Azure Service Bus
// （连接串指向真实的命名空间，或本地 Emulator）。
func ExampleNewServer() {
	asbSrv := azuresb.NewServer(
		azuresb.WithConnectionString(
			"Endpoint=sb://127.0.0.1;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=key",
		),
		azuresb.WithCodec("json"),
	)

	_ = azuresb.RegisterSubscriber(asbSrv, "test_topic", handleHygrothermograph)

	app := kratos.New(
		kratos.Name("azuresb"),
		kratos.Server(
			asbSrv,
		),
	)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}
