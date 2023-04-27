module github.com/tx7do/kratos-transport/broker/nats

go 1.19

replace github.com/tx7do/kratos-transport => ../../

require (
	github.com/go-kratos/kratos/v2 v2.6.1
	github.com/nats-io/nats.go v1.25.0
	github.com/stretchr/testify v1.8.2
	github.com/tx7do/kratos-transport v1.0.5
	go.opentelemetry.io/otel v1.14.0
	go.opentelemetry.io/otel/trace v1.14.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/nats-io/nats-server/v2 v2.9.6 // indirect
	github.com/nats-io/nkeys v0.4.4 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/openzipkin/zipkin-go v0.4.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/otel/exporters/jaeger v1.14.0 // indirect
	go.opentelemetry.io/otel/exporters/zipkin v1.14.0 // indirect
	go.opentelemetry.io/otel/sdk v1.14.0 // indirect
	golang.org/x/crypto v0.8.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
