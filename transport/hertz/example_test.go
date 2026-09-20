package hertz_test

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/transport/hertz"
)

type HelloResponse struct {
	Message string `json:"message"`
}

// ExampleNewServer 演示启动一个 hertz HTTP 服务：
// 注册带路径参数的路由，返回 JSON 格式的问候语。
// 没有 Output 注释，go test 只编译不执行；纯 HTTP 服务，可独立运行。
func ExampleNewServer() {
	srv := hertz.NewServer(
		hertz.WithAddress("127.0.0.1:8800"),
	)

	srv.GET("/hello/:name", func(ctx context.Context, c *app.RequestContext) {
		c.JSON(200, &HelloResponse{Message: "Hello " + c.Param("name")})
	})

	app := kratos.New(
		kratos.Name("hertz"),
		kratos.Server(
			srv,
		),
	)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}
