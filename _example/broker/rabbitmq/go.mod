module github.com/tx7do/kratos-transport/_example/broker/rabbitmq

go 1.20

require (
	github.com/tx7do/kratos-transport v1.1.3
	github.com/tx7do/kratos-transport/broker/rabbitmq v1.2.6
)

require (
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/go-kratos/kratos/v2 v2.7.2 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.19.1 // indirect
	github.com/openzipkin/zipkin-go v0.4.2 // indirect
	github.com/rabbitmq/amqp091-go v1.9.0 // indirect
	go.opentelemetry.io/otel v1.24.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.24.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.24.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.24.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.24.0 // indirect
	go.opentelemetry.io/otel/exporters/zipkin v1.24.0 // indirect
	go.opentelemetry.io/otel/metric v1.24.0 // indirect
	go.opentelemetry.io/otel/sdk v1.24.0 // indirect
	go.opentelemetry.io/otel/trace v1.24.0 // indirect
	go.opentelemetry.io/proto/otlp v1.1.0 // indirect
	golang.org/x/net v0.22.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240308144416-29370a3891b7 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240308144416-29370a3891b7 // indirect
	google.golang.org/grpc v1.62.1 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)

replace github.com/tx7do/kratos-transport => ../../../

replace github.com/tx7do/kratos-transport/broker/rabbitmq => ../../../broker/rabbitmq
