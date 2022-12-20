module github.com/tx7do/kratos-transport/transport/kafka

go 1.19

replace github.com/tx7do/kratos-transport => ../../

require (
	github.com/go-kratos/kratos/v2 v2.5.3
	github.com/segmentio/kafka-go v0.4.38
	github.com/stretchr/testify v1.8.1
	github.com/tx7do/kratos-transport v1.0.5
	github.com/tx7do/kratos-transport/broker/kafka v0.0.0-20221220104642-e974d66866a9
	go.opentelemetry.io/otel v1.11.2
	go.opentelemetry.io/otel/trace v1.11.2
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-playground/form/v4 v4.2.0 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/klauspost/compress v1.15.13 // indirect
	github.com/openzipkin/zipkin-go v0.4.1 // indirect
	github.com/pierrec/lz4/v4 v4.1.17 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/xdg/scram v1.0.5 // indirect
	github.com/xdg/stringprep v1.0.3 // indirect
	go.opentelemetry.io/otel/exporters/jaeger v1.11.2 // indirect
	go.opentelemetry.io/otel/exporters/zipkin v1.11.2 // indirect
	go.opentelemetry.io/otel/sdk v1.11.2 // indirect
	golang.org/x/crypto v0.4.0 // indirect
	golang.org/x/sys v0.3.0 // indirect
	golang.org/x/text v0.5.0 // indirect
	google.golang.org/genproto v0.0.0-20221207170731-23e4bf6bdc37 // indirect
	google.golang.org/grpc v1.51.0 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
