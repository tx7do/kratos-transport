module github.com/tx7do/kratos-transport/transport/graphql

go 1.19

replace github.com/tx7do/kratos-transport => ../../

replace google.golang.org/grpc => google.golang.org/grpc v1.46.2

require (
	github.com/99designs/gqlgen v0.17.33
	github.com/go-kratos/kratos/v2 v2.6.2
	github.com/gorilla/mux v1.8.0
	github.com/tx7do/kratos-transport v1.0.6
)

require (
	github.com/agnivade/levenshtein v1.1.1 // indirect
	github.com/go-playground/form/v4 v4.2.0 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.3 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/vektah/gqlparser/v2 v2.5.3 // indirect
	golang.org/x/net v0.8.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230530153820-e85fd2cbaebc // indirect
	google.golang.org/grpc v1.56.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
