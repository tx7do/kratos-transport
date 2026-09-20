package graphql_test

import (
	"context"
	"math/rand"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	api "github.com/tx7do/kratos-transport/testing/api/graphql"
	"github.com/tx7do/kratos-transport/transport/graphql"
)

type resolver struct{}

func (r *resolver) Query() api.QueryResolver {
	return &queryResolver{}
}

type queryResolver struct{}

func (r *queryResolver) Hygrothermograph(_ context.Context) (*api.Hygrothermograph, error) {
	return &api.Hygrothermograph{
		Humidity:    float64(rand.Intn(100)),
		Temperature: float64(rand.Intn(100)),
	}, nil
}

// ExampleNewServer 演示启动一个 GraphQL 查询服务：
// 注册温湿度查询解析器，并把生成的 schema 挂载到 /query 端点。
// 没有 Output 注释，go test 只编译不执行；实际运行会阻塞等待进程退出信号。
func ExampleNewServer() {
	gqlSrv := graphql.NewServer(
		graphql.WithAddress(":8800"),
	)

	gqlSrv.Handle("/query", api.NewExecutableSchema(api.Config{Resolvers: &resolver{}}))

	app := kratos.New(
		kratos.Name("graphql"),
		kratos.Server(
			gqlSrv,
		),
	)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}
