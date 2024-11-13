module github.com/tx7do/kratos-transport/transport/thrift

go 1.22.7

toolchain go1.23.3

require (
	github.com/apache/thrift v0.21.0
	github.com/go-kratos/kratos/v2 v2.8.2
	github.com/tx7do/kratos-transport v1.1.12
)

require (
	github.com/go-playground/form/v4 v4.2.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	google.golang.org/protobuf v1.35.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/tx7do/kratos-transport => ../../
