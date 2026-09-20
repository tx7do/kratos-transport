package gcpubsub_test

import (
	"context"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/broker"
	api "github.com/tx7do/kratos-transport/testing/api/manual"
	"github.com/tx7do/kratos-transport/transport/gcpubsub"
)

func handleHygrothermograph(_ context.Context, topic string, headers broker.Headers, msg *api.Hygrothermograph) error {
	log.Infof("Topic %s, Headers: %+v, Payload: %+v\n", topic, headers, msg)
	return nil
}

// ExampleNewServer 演示把 Google Cloud Pub/Sub 订阅挂到 kratos 应用上。
// 没有 Output 注释，go test 只编译不执行；实际运行需要本地 Pub/Sub 模拟器
// (localhost:8085) 或真实 GCP 项目凭据。
func ExampleNewServer() {
	gcsSrv := gcpubsub.NewServer(
		gcpubsub.WithProjectID("my-gcp-project"),
		gcpubsub.WithEndpoint("localhost:8085"),
		gcpubsub.WithCodec("json"),
	)

	_ = gcpubsub.RegisterSubscriber(gcsSrv, "test_topic", handleHygrothermograph)

	app := kratos.New(
		kratos.Name("gcpubsub"),
		kratos.Server(
			gcsSrv,
		),
	)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}
