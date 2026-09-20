package keepalive_test

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/transport/keepalive"
)

// ExampleNewServer 演示启动一个 gRPC 健康检查（keepalive）服务：
// 内置 grpc_health_v1 健康检查协议，通常作为 cron/machinery/mcp 等
// 无对外端口的服务的心跳保活组件挂到 kratos 应用上。
// 没有 Output 注释，go test 只编译不执行；实际运行会阻塞等待进程退出信号。
func ExampleNewServer() {
	// 不指定地址时默认监听本机 IP + 随机端口，
	// 可用 SetKeepAliveHost/SetKeepAliveInterface 固定注册的主机。
	keepalive.SetKeepAliveHost("127.0.0.1")

	kaSrv := keepalive.NewServer(
		keepalive.WithServiceKind(keepalive.KindKeepAlive),
		keepalive.WithAddress("127.0.0.1:19000"),
	)

	ep, err := kaSrv.Endpoint()
	if err != nil {
		log.Error(err)
		return
	}
	log.Infof("keepalive endpoint: %s", ep.String())

	app := kratos.New(
		kratos.Name("keepalive"),
		kratos.Server(
			kaSrv,
		),
	)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}
