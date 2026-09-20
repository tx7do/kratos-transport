package thrift_test

import (
	"context"
	"math/rand"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	api "github.com/tx7do/kratos-transport/testing/api/thrift/gen-go/hygrothermograph"
	"github.com/tx7do/kratos-transport/transport/thrift"
)

type hygrothermographHandler struct{}

func (h *hygrothermographHandler) GetHygrothermograph(_ context.Context) (*api.Hygrothermograph, error) {
	humidity := float64(rand.Intn(100))
	temperature := float64(rand.Intn(100))
	log.Infof("Humidity: %v, Temperature: %v", humidity, temperature)
	return &api.Hygrothermograph{
		Humidity:    &humidity,
		Temperature: &temperature,
	}, nil
}

// ExampleNewServer 演示启动一个 Thrift RPC 服务：
// 以二进制协议注册 HygrothermographService 处理器，提供温湿度查询 RPC。
// 没有 Output 注释，go test 只编译不执行；实际运行会阻塞等待进程退出信号。
func ExampleNewServer() {
	srv := thrift.NewServer(
		thrift.WithAddress(":7700"),
		thrift.WithProcessor(api.NewHygrothermographServiceProcessor(&hygrothermographHandler{})),
		thrift.WithProtocol("binary"),
	)

	app := kratos.New(
		kratos.Name("thrift"),
		kratos.Server(
			srv,
		),
	)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}
