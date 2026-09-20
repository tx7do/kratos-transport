package cron_test

import (
	"context"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/transport/cron"
)

// ExampleNewServer 演示启动一个 cron 定时任务服务：
// 通过 kratos.AfterStart 在服务启动后注册一个每 10 秒执行的任务和一个每分钟执行的任务。
// StartTimerJob 要求服务已启动，因此注册动作放在 AfterStart 钩子里。
// 没有 Output 注释，go test 只编译不执行；实际运行会阻塞等待进程退出信号。
func ExampleNewServer() {
	srv := cron.NewServer(
		cron.WithEnableKeepAlive(false),
	)

	app := kratos.New(
		kratos.Name("cron"),
		kratos.Server(
			srv,
		),
		kratos.AfterStart(func(_ context.Context) error {
			// 每 10 秒执行一次（支持秒级表达式：秒 分 时 日 月 周）
			if _, err := srv.StartTimerJob("*/10 * * * * *", func() {
				log.Info("task run every 10 seconds")
			}); err != nil {
				log.Error(err)
			}

			// 每分钟执行一次
			if _, err := srv.StartTimerJob("0 */1 * * * *", func() {
				log.Info("task run every minute")
			}); err != nil {
				log.Error(err)
			}

			return nil
		}),
	)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}
