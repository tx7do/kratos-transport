package kafka_test

import (
	"context"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/broker"
	api "github.com/tx7do/kratos-transport/testing/api/manual"
	"github.com/tx7do/kratos-transport/tracing"
	"github.com/tx7do/kratos-transport/transport/kafka"
)

func handleHygrothermograph(_ context.Context, topic string, headers broker.Headers, msg *api.Hygrothermograph) error {
	log.Infof("Topic %s, Headers: %+v, Payload: %+v\n", topic, headers, msg)
	return nil
}

// ExampleNewServer 演示把 Kafka 订阅挂到 kratos 应用上，并接入 OpenTelemetry 链路追踪。
// 没有 Output 注释，go test 只编译不执行；实际运行需要本地 Kafka (localhost:9092)。
func ExampleNewServer() {
	ctx := context.Background()

	kafkaSrv := kafka.NewServer(
		kafka.WithAddress([]string{"localhost:9092"}),
		kafka.WithCodec("json"),
		kafka.WithBrokerOptions(
			broker.WithTracerProvider(
				tracing.NewTracerProvider("otlp-grpc", "localhost:4317", "tracer_tester", "", "1.0.0", 1.0),
			),
		),
	)

	_ = kafka.RegisterSubscriber(kafkaSrv, ctx, "test_topic", "a-group", false, handleHygrothermograph)

	app := kratos.New(
		kratos.Name("kafka"),
		kratos.Server(
			kafkaSrv,
		),
	)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}
