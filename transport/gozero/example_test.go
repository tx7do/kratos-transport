package gozero_test

import (
	"net/http"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"

	"github.com/tx7do/kratos-transport/transport/gozero"
)

type HelloResponse struct {
	Message string `json:"message"`
}

// ExampleNewServer 演示启动一个 go-zero rest HTTP 服务：
// 通过 AddRoute 注册带路径参数的路由，返回 JSON 格式的问候语。
// 没有 Output 注释，go test 只编译不执行；纯 HTTP 服务，可独立运行。
func ExampleNewServer() {
	srv := gozero.NewServer(
		gozero.WithAddress(":8800"),
	)

	srv.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/hello/:name",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			httpx.OkJson(w, &HelloResponse{Message: "Hello " + pathvar.Vars(r)["name"]})
		},
	})

	app := kratos.New(
		kratos.Name("gozero"),
		kratos.Server(
			srv,
		),
	)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}
