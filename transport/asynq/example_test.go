package asynq_test

import (
	"time"

	"github.com/hibiken/asynq"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	asynqServer "github.com/tx7do/kratos-transport/transport/asynq"
)

const (
	localRedisURI = "redis://:*Abcd123456@127.0.0.1:6379/1"

	testTask1        = "test_task_1"
	testDelayTask    = "test_delay_task"
	testPeriodicTask = "test_periodic_task"
)

type TaskPayload struct {
	Message string `json:"message"`
}

// ExampleNewServer 演示基于 asynq 的延迟任务/周期任务：
// 注册任务处理器，投递一个立即任务、一个延迟任务和一个每分钟执行的周期任务。
// 没有 Output 注释，go test 只编译不执行；实际运行需要本地 Redis 并开启 db 1。
func ExampleNewServer() {
	redisConnOpt, err := asynq.ParseRedisURI(localRedisURI)
	if err != nil {
		log.Error(err)
		return
	}

	srv := asynqServer.NewServer(
		asynqServer.WithRedisConnOpt(redisConnOpt),
		asynqServer.WithShutdownTimeout(3*time.Second),
		asynqServer.WithConcurrency(10),
	)

	_ = asynqServer.RegisterSubscriber(srv, testTask1, handleTask1)
	_ = asynqServer.RegisterSubscriber(srv, testDelayTask, handleDelayTask)
	_ = asynqServer.RegisterSubscriber(srv, testPeriodicTask, handlePeriodicTask)

	// 立即任务：最多重试3次，10秒超时，20秒后过期
	_ = srv.NewTask(
		testTask1,
		&TaskPayload{Message: "immediate task"},
		asynq.MaxRetry(3),
		asynq.Timeout(10*time.Second),
		asynq.Deadline(time.Now().Add(20*time.Second)),
		asynq.TaskID(testTask1),
	)

	// 延迟任务：3秒后执行
	_ = srv.NewTask(
		testDelayTask,
		&TaskPayload{Message: "delay task"},
		asynq.ProcessIn(3*time.Second),
		asynq.TaskID(testDelayTask),
	)

	// 周期性任务：每分钟执行一次
	if _, err := srv.NewPeriodicTask(
		"*/1 * * * ?",
		testPeriodicTask+"1",
		&TaskPayload{Message: "periodic task 1"},
	); err != nil {
		log.Error(err)
	}

	app := kratos.New(
		kratos.Name("asynq"),
		kratos.Server(
			srv,
		),
	)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}

func handleTask1(taskType string, taskData *TaskPayload) error {
	log.Infof("[%s] Task Type: [%s], Payload: [%s]", time.Now().Format("2006-01-02 15:04:05"), taskType, taskData.Message)
	return nil
}

func handleDelayTask(taskType string, taskData *TaskPayload) error {
	log.Infof("[%s] Delay Task Type: [%s], Payload: [%s]", time.Now().Format("2006-01-02 15:04:05"), taskType, taskData.Message)
	return nil
}

func handlePeriodicTask(taskType string, taskData *TaskPayload) error {
	log.Infof("[%s] Periodic Task Type: [%s], Payload: [%s]", time.Now().Format("2006-01-02 15:04:05"), taskType, taskData.Message)
	return nil
}
