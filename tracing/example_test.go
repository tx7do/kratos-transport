package tracing_test

import (
	"context"
	"log"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/tx7do/kratos-transport/tracing"
)

// ExampleNewTracerProvider 演示创建 OpenTelemetry TracerProvider，
// 并用它创建生产者 Tracer 做手工埋点。
// 没有 Output 注释，go test 只编译不执行；实际导出需要可访问的 OTLP Collector (localhost:4317)。
func ExampleNewTracerProvider() {
	// 创建 TracerProvider：OTLP gRPC 导出到 localhost:4317，
	// serviceName/instanceId/version 写入资源属性，采样率 1.0 表示全采样
	tp := tracing.NewTracerProvider("otlp-grpc", "localhost:4317", "demo-service", "", "1.0.0", 1.0)
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Println(err)
		}
	}()

	// exporter 名称支持：otlp-grpc、otlp-http、stdout（zipkin/jaeger 已不再支持）；
	// endpoint 传空则不注册导出器，也可以用 tracing.NewExporter 自建导出器后
	// 通过 tracing.WithTracerProvider 自行组装 TracerProvider。

	// 基于该 Provider 创建一个生产者 Tracer，向 carrier 注入 trace 上下文
	tracer := tracing.NewTracer(trace.SpanKindProducer, "demo-producer",
		tracing.WithTracerProvider(tp),
		tracing.WithTracerName("demo-tracer"),
	)

	carrier := propagation.MapCarrier{}
	ctx, span := tracer.Start(context.Background(), carrier)
	tracer.End(ctx, span, nil)

	// 在 kratos-transport 的 broker/transport 中接入链路追踪时，
	// 通过 broker.WithTracerProvider(tp) 注入，例如：
	//   kafka.NewServer(
	//       kafka.WithAddress([]string{"localhost:9092"}),
	//       kafka.WithCodec("json"),
	//       kafka.WithBrokerOptions(broker.WithTracerProvider(tp)),
	//   )
}
