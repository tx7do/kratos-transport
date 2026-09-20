package machinery_test

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/transport/machinery"
)

const (
	localRedisAddr = "123456@localhost:6379"

	testImmediateTask = "test_immediate_task"
	testDelayTask     = "test_delay_task"
	testPeriodicTask  = "test_periodic_task"
)

func handleImmediateTask() error {
	log.Info("run immediate task")
	return nil
}

func handleDelayTask() error {
	log.Info("run delay task")
	return nil
}

func handlePeriodicTask() error {
	log.Info("run periodic task")
	return nil
}

// ExampleNewServer 演示基于 machinery 的异步任务队列：
// 注册任务处理器，投递一个立即任务、一个延迟任务和一个每分钟执行的周期任务。
// 没有 Output 注释，go test 只编译不执行；实际运行需要本地 Redis 并阻塞等待进程退出信号。
func ExampleNewServer() {
	ctx := context.Background()

	srv := machinery.NewServer(
		machinery.WithBrokerAddress(localRedisAddr, 0, machinery.BrokerTypeRedis),
		machinery.WithResultBackendAddress(localRedisAddr, 0, machinery.BackendTypeRedis),
	)

	for name, handler := range map[string]any{
		testImmediateTask: handleImmediateTask,
		testDelayTask:     handleDelayTask,
		testPeriodicTask:  handlePeriodicTask,
	} {
		if err := srv.HandleFunc(name, handler); err != nil {
			log.Error(err)
			return
		}
	}

	// 立即任务
	if err := srv.NewTask(ctx, testImmediateTask); err != nil {
		log.Error(err)
	}

	// 延迟任务：5 秒后执行
	if err := srv.NewTask(ctx, testDelayTask,
		machinery.WithDelayTime(time.Now().UTC().Add(5*time.Second)),
	); err != nil {
		log.Error(err)
	}

	// 周期任务：每分钟一次（周期任务最大精度只到分钟）
	if err := srv.NewPeriodicTask(ctx, "*/1 * * * ?", testPeriodicTask); err != nil {
		log.Error(err)
	}

	app := kratos.New(
		kratos.Name("machinery"),
		kratos.Server(
			srv,
		),
	)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}
