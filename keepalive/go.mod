module github.com/tx7do/kratos-transport/keepalive

go 1.23.0

toolchain go1.24.3

replace github.com/tx7do/kratos-transport => ../

require (
	github.com/go-kratos/kratos/v2 v2.8.4
	github.com/stretchr/testify v1.10.0
	google.golang.org/grpc v1.72.2
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250603155806-513f23925822 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
