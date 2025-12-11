module github.com/tx7do/kratos-transport/transport/keepalive

go 1.24.0

toolchain go1.24.3

replace github.com/tx7do/kratos-transport => ../../

require (
	github.com/go-kratos/kratos/v2 v2.9.2
	github.com/stretchr/testify v1.11.1
	github.com/tx7do/kratos-transport v1.1.17
	google.golang.org/grpc v1.77.0
)
