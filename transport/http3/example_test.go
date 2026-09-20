package http3_test

import (
	"net/http"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/transport/http3"
)

// ExampleNewServer 演示启动一个 HTTP/3 (QUIC) 服务：
// 未提供证书时自动生成自签名证书，经 Router 注册带路径参数的路由。
// 没有 Output 注释，go test 只编译不执行；纯 HTTP/3 服务，可独立运行（客户端需信任自签名证书）。
func ExampleNewServer() {
	srv := http3.NewServer(
		http3.WithAddress(":8800"),
	)

	router := srv.Route("/api")
	router.GET("/hello/{name}", func(ctx http3.Context) error {
		return ctx.String(http.StatusOK, "Hello "+ctx.Vars().Get("name"))
	})

	app := kratos.New(
		kratos.Name("http3"),
		kratos.Server(
			srv,
		),
	)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}
