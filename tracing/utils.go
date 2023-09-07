package tracing

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/zipkin"

	"go.opentelemetry.io/otel/sdk/resource"
	traceSdk "go.opentelemetry.io/otel/sdk/trace"

	semConv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

// NewExporter 创建一个导出器，支持：jaeger和zipkin
func NewExporter(exporterName, endpoint string) (traceSdk.SpanExporter, error) {
	switch exporterName {
	case "zipkin":
		return NewZipkinExporter(endpoint)
	case "jaeger":
		fallthrough
	case "otlptracehttp":
		return NewOtlpHttpExporter(endpoint)
	case "otlptracegrpc":
		return NewOtlpGrpcExporter(endpoint)
	case "prometheus":
		return NewPrometheusExporter(endpoint)
	default:
		return nil, errors.New("exporter type not support")
	}
}

// NewJaegerExporter 创建一个jaeger导出器
//func NewJaegerExporter(endpoint string) (traceSdk.SpanExporter, error) {
//	return jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(endpoint)))
//}

// NewZipkinExporter 创建一个zipkin导出器
func NewZipkinExporter(endpoint string) (traceSdk.SpanExporter, error) {
	return zipkin.New(endpoint)
}

// NewOtlpHttpExporter 创建一个OTLP HTTP导出器
func NewOtlpHttpExporter(endpoint string) (traceSdk.SpanExporter, error) {
	return otlptracehttp.New(context.Background(), otlptracehttp.WithEndpoint(endpoint))
}

// NewOtlpGrpcExporter 创建一个OTLP GRPC导出器
func NewOtlpGrpcExporter(endpoint string) (traceSdk.SpanExporter, error) {
	return otlptracegrpc.New(context.Background(), otlptracegrpc.WithEndpoint(endpoint))
}

// NewTracerProvider 创建一个链路追踪器
func NewTracerProvider(exporterName, endpoint, serviceName, instanceId, version string, sampler float64) *traceSdk.TracerProvider {
	if instanceId == "" {
		ud, _ := uuid.NewUUID()
		instanceId = ud.String()
	}
	if version == "" {
		version = "x.x.x"
	}

	opts := []traceSdk.TracerProviderOption{
		traceSdk.WithSampler(traceSdk.ParentBased(traceSdk.TraceIDRatioBased(sampler))),
		traceSdk.WithResource(resource.NewSchemaless(
			semConv.ServiceNameKey.String(serviceName),
			semConv.ServiceInstanceIDKey.String(instanceId),
			semConv.ServiceVersionKey.String(version),
		)),
	}

	if len(endpoint) > 0 {
		exp, err := NewExporter(exporterName, endpoint)
		if err != nil {
			panic(err)
		}

		opts = append(opts, traceSdk.WithBatcher(exp))
	}

	return traceSdk.NewTracerProvider(opts...)
}
