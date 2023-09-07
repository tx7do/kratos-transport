module github.com/tx7do/kratos-transport/transport/rocketmq

go 1.19


require (
	github.com/go-kratos/kratos/v2 v2.7.0
	github.com/stretchr/testify v1.8.4
	github.com/tx7do/kratos-transport v1.0.11
	github.com/tx7do/kratos-transport/broker/rocketmq v0.0.0-20230907091654-32334115fdef
	go.opentelemetry.io/otel v1.17.0
	go.opentelemetry.io/otel/trace v1.17.0
)

require (
	github.com/aliyunmq/mq-http-go-sdk v1.0.3 // indirect
	github.com/andybalholm/brotli v1.0.5 // indirect
	github.com/apache/rocketmq-client-go/v2 v2.1.1 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-playground/form/v4 v4.2.1 // indirect
	github.com/gogap/errors v0.0.0-20210818113853-edfbba0ddea9 // indirect
	github.com/gogap/stack v0.0.0-20150131034635-fef68dddd4f8 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/uuid v1.3.1 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.17.1 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.16.7 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/openzipkin/zipkin-go v0.4.2 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/stathat/consistent v1.0.0 // indirect
	github.com/tidwall/gjson v1.16.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.49.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.17.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.17.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.17.0 // indirect
	go.opentelemetry.io/otel/exporters/zipkin v1.17.0 // indirect
	go.opentelemetry.io/otel/metric v1.17.0 // indirect
	go.opentelemetry.io/otel/sdk v1.17.0 // indirect
	go.opentelemetry.io/proto/otlp v1.0.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/net v0.15.0 // indirect
	golang.org/x/sys v0.12.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230822172742-b8732ec3820d // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230822172742-b8732ec3820d // indirect
	google.golang.org/grpc v1.58.0 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/tx7do/kratos-transport => ../../

replace github.com/tx7do/kratos-transport/broker/rocketmq => ../../broker/rocketmq
