module github.com/tx7do/kratos-transport/broker/rabbitmq

go 1.19

require (
	github.com/go-kratos/kratos/v2 v2.7.0
	github.com/google/uuid v1.3.1
	github.com/rabbitmq/amqp091-go v1.8.1
	github.com/stretchr/testify v1.8.4
	github.com/tx7do/kratos-transport v1.0.8
	go.opentelemetry.io/otel v1.17.0
	go.opentelemetry.io/otel/trace v1.17.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/openzipkin/zipkin-go v0.4.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/otel/exporters/jaeger v1.17.0 // indirect
	go.opentelemetry.io/otel/exporters/zipkin v1.17.0 // indirect
	go.opentelemetry.io/otel/metric v1.17.0 // indirect
	go.opentelemetry.io/otel/sdk v1.17.0 // indirect
	golang.org/x/sys v0.12.0 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/tx7do/kratos-transport => ../../
