package mqtt_test

import (
	"context"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/broker"
	api "github.com/tx7do/kratos-transport/testing/api/manual"
	"github.com/tx7do/kratos-transport/transport/mqtt"
)

func handleHygrothermograph(_ context.Context, topic string, headers broker.Headers, msg *api.Hygrothermograph) error {
	log.Infof("Topic %s, Headers: %+v, Payload: %+v\n", topic, headers, msg)
	return nil
}

// ExampleNewServer 演示把 MQTT 订阅挂到 kratos 应用上。
// 没有 Output 注释，go test 只编译不执行；实际运行需要可连的 MQTT broker。
func ExampleNewServer() {
	ctx := context.Background()

	mqttSrv := mqtt.NewServer(
		mqtt.WithAddress([]string{"tcp://broker-cn.emqx.io:1883"}),
		mqtt.WithCodec("json"),
	)

	_ = mqtt.RegisterSubscriber(mqttSrv, ctx, "topic/bobo/#", handleHygrothermograph)

	app := kratos.New(
		kratos.Name("mqtt"),
		kratos.Server(
			mqttSrv,
		),
	)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}
