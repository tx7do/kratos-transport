package gin_test

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	kGin "github.com/tx7do/kratos-transport/transport/gin"
)

type HelloResponse struct {
	Message string `json:"message"`
}

// ExampleNewServer 演示启动一个 gin HTTP 服务：
// 注册带路径参数的路由，返回 JSON 格式的问候语。
// 没有 Output 注释，go test 只编译不执行；纯 HTTP 服务，可独立运行。
func ExampleNewServer() {
	srv := kGin.NewServer(
		kGin.WithAddress(":8800"),
	)

	srv.GET("/hello/:name", func(c *gin.Context) {
		c.JSON(http.StatusOK, &HelloResponse{Message: "Hello " + c.Param("name")})
	})

	app := kratos.New(
		kratos.Name("gin"),
		kratos.Server(
			srv,
		),
	)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}
