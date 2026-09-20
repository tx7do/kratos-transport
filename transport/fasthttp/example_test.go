package fasthttp_test

import (
	"encoding/json"

	"github.com/valyala/fasthttp"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	kFasthttp "github.com/tx7do/kratos-transport/transport/fasthttp"
)

type Hygrothermograph struct {
	Humidity    string `json:"humidity"`
	Temperature string `json:"temperature"`
}

// ExampleNewServer 演示启动一个 fasthttp HTTP 服务：
// 注册 GET 路由，返回 JSON 格式的温湿度数据。
// 没有 Output 注释，go test 只编译不执行；纯 HTTP 服务，可独立运行。
func ExampleNewServer() {
	srv := kFasthttp.NewServer(
		kFasthttp.WithAddress(":8800"),
	)

	srv.GET("/hygrothermograph", func(c *fasthttp.RequestCtx) {
		out := Hygrothermograph{Humidity: "68", Temperature: "27"}
		_ = json.NewEncoder(c.Response.BodyWriter()).Encode(&out)
	})

	app := kratos.New(
		kratos.Name("fasthttp"),
		kratos.Server(
			srv,
		),
	)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}
