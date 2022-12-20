package main

import (
	"context"
	"github.com/tx7do/kratos-transport/tracing"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/transport/kafka"

	api "github.com/tx7do/kratos-transport/_example/api/manual"
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
	case "jaeger":
		return broker.WithTracerProvider(tracing.NewTracerProvider(exporterName,
			"http://localhost:14268/api/traces",
			serviceName,
			"",
			"1.0.0",
			1.0,
		),
			"kafka-tracer",
		)
	case "zipkin":
		return broker.WithTracerProvider(tracing.NewTracerProvider(exporterName,
			"http://localhost:9411/api/v2/spans",
			serviceName,
			"test",
			"1.0.0",
			1.0,
		),
			"kafka-tracer",
		)
	}

	return nil
}

func main() {
	ctx := context.Background()

	kafkaSrv := kafka.NewServer(
		kafka.WithAddress([]string{testBrokers}),
		kafka.WithCodec("json"),
		kafka.WithBrokerOptions(createTracerProvider("jaeger", "tracer_tester")),
	)

	_ = kafkaSrv.RegisterSubscriber(ctx,
		testTopic, testGroupId, false,
		api.RegisterHygrothermographHandler(handleHygrothermograph),
		api.HygrothermographCreator,
	)

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
