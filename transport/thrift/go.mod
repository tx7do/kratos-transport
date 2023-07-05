module github.com/tx7do/kratos-transport/transport/thrift

go 1.19

replace github.com/tx7do/kratos-transport => ../../

require (
	github.com/apache/thrift v0.18.1
	github.com/go-kratos/kratos/v2 v2.6.3
	github.com/tx7do/kratos-transport v1.0.7
)

require (
	github.com/go-playground/form/v4 v4.2.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
