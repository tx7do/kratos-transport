package hptimer_test

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/transport/hptimer"
)

type heartbeatPayload struct {
	Service string `json:"service"`
	Seq     int    `json:"seq"`
}

// ExampleNewServer 演示启动一个高精度定时器服务：
// 通过 kratos.AfterStart 在服务启动后注册一个每 500 毫秒触发一次的心跳任务，
// 任务通过 NewTimerTask 创建，负载为示例内定义的本地 struct。
// 没有 Output 注释，go test 只编译不执行；实际运行会阻塞等待进程退出信号。
func ExampleNewServer() {
	srv := hptimer.NewServer(
		hptimer.WithGracefullyShutdown(true),
	)

	app := kratos.New(
		kratos.Name("hptimer"),
		kratos.Server(
			srv,
		),
		kratos.AfterStart(func(_ context.Context) error {
			// AddTask 要求服务已启动，因此注册动作放在 AfterStart 钩子里
			task := hptimer.NewTimerTask(
				"heartbeat",
				time.Now().Add(500*time.Millisecond),
				hptimer.WithInterval(500*time.Millisecond),
				hptimer.WithData(heartbeatPayload{Service: "hptimer", Seq: 1}),
				hptimer.WithCallback(func(_ context.Context) error {
					log.Info("heartbeat triggered")
					return nil
				}),
			)

			if srv.AddTask(task) == "" {
				log.Error("add timer task failed")
			}

			return nil
		}),
	)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}
