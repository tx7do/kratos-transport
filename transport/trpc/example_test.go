package trpc_test

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/transport/trpc"
)

// ExampleNewServer 演示把 tRPC 服务挂到 kratos 应用上：
// 监听 :8972 并注册服务名；业务的 trpc service 需在服务启动后
// 通过 AddService 按生成的 tRPC 服务骨架挂载。
// 没有 Output 注释，go test 只编译不执行；实际运行需要 tRPC 全局配置并阻塞等待进程退出信号。
func ExampleNewServer() {
	srv := trpc.NewServer(
		trpc.WithAddress(":8972"),
		trpc.WithNamespace("Development"),
		trpc.WithEnvName("dev"),
		trpc.WithServiceName("trpc.kratos.transport.demo"),
	)

	ep, err := srv.Endpoint()
	if err != nil {
		log.Error(err)
		return
	}
	log.Infof("tRPC endpoint: %s", ep.String())

	app := kratos.New(
		kratos.Name("trpc"),
		kratos.Server(
			srv,
		),
	)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}
