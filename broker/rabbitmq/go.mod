module github.com/tx7do/kratos-transport/broker/rabbitmq

go 1.19

replace github.com/tx7do/kratos-transport => ../../

require (
	github.com/go-kratos/kratos/v2 v2.5.3
	github.com/google/uuid v1.3.0
	github.com/streadway/amqp v1.0.0
	github.com/stretchr/testify v1.8.1
	github.com/tx7do/kratos-transport v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/otel v1.11.1
	go.opentelemetry.io/otel/trace v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/openzipkin/zipkin-go v0.4.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/otel/exporters/jaeger v1.9.0 // indirect
	go.opentelemetry.io/otel/exporters/zipkin v1.9.0 // indirect
	go.opentelemetry.io/otel/sdk v1.9.0 // indirect
	golang.org/x/sys v0.2.0 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
