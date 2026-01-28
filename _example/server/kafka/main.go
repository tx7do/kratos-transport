package main

import (
	"context"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/tracing"
	"github.com/tx7do/kratos-transport/transport/kafka"

	api "github.com/tx7do/kratos-transport/testing/api/manual"
)

const (
	testBrokers = "localhost:9092"
	testTopic   = "test_topic"
	testGroupId = "a-group"
)

func handleHygrothermograph(_ context.Context, topic string, headers broker.Headers, msg *api.Hygrothermograph) error {
	log.Infof("Topic %s, Headers: %+v, Payload: %+v\n", topic, headers, msg)
	return nil
}

func createTracerProvider(exporterName, serviceName string) broker.Option {
	switch exporterName {
	case "otlp-grpc":
		return broker.WithTracerProvider(
			tracing.NewTracerProvider(exporterName,
				"localhost:4317",
				serviceName,
				"",
				"1.0.0",
				1.0,
			),
		)
	case "zipkin":
		return broker.WithTracerProvider(
			tracing.NewTracerProvider(exporterName,
				"http://localhost:9411/api/v2/spans",
				serviceName,
				"test",
				"1.0.0",
				1.0,
			),
		)
	}

	return nil
}

func main() {
	ctx := context.Background()

	kafkaSrv := kafka.NewServer(
		kafka.WithAddress([]string{testBrokers}),
		kafka.WithCodec("json"),
		kafka.WithBrokerOptions(createTracerProvider("otlp-grpc", "tracer_tester")),
	)

	_ = kafka.RegisterSubscriber(kafkaSrv, ctx, testTopic, testGroupId, false, handleHygrothermograph)

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
