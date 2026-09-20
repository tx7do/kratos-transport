package iris_test

import (
	"github.com/kataras/iris/v12"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	kIris "github.com/tx7do/kratos-transport/transport/iris"
)

type Hygrothermograph struct {
	Humidity    string `json:"humidity"`
	Temperature string `json:"temperature"`
}

// ExampleNewServer 演示启动一个 iris HTTP 服务：
// 注册带路径参数的路由返回问候语，再注册一个 JSON 路由。
// 没有 Output 注释，go test 只编译不执行；纯 HTTP 服务，可独立运行。
func ExampleNewServer() {
	srv := kIris.NewServer(
		kIris.WithAddress(":8800"),
	)

	srv.Get("/hello/{name}", func(ctx iris.Context) {
		_, _ = ctx.WriteString("Hello " + ctx.Params().Get("name"))
	})

	srv.Get("/hygrothermograph", func(ctx iris.Context) {
		out := Hygrothermograph{Humidity: "68", Temperature: "27"}
		_ = ctx.JSON(&out)
	})

	app := kratos.New(
		kratos.Name("iris"),
		kratos.Server(
			srv,
		),
	)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}
