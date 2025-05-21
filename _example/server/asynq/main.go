package main

import (
	"github.com/hibiken/asynq"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	asynqServer "github.com/tx7do/kratos-transport/transport/asynq"
)

var testServer *asynqServer.Server

const (
	localRedisAddr = "127.0.0.1:6379"
	redisPassword  = "*Abcd123456"

	testTask1        = "test_task_1"
	testDelayTask    = "test_delay_task"
	testPeriodicTask = "test_periodic_task"
)

type TaskPayload struct {
	Message string `json:"message"`
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

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	srv := asynqServer.NewServer(
		asynqServer.WithAddress(localRedisAddr),
		asynqServer.WithRedisPassword(redisPassword),
		asynqServer.WithRedisDatabase(1),
		asynqServer.WithShutdownTimeout(3*time.Second),
		asynqServer.WithConcurrency(10),
	)

	testServer = srv

	app := kratos.New(
		kratos.Name("asynq"),
		kratos.Server(
			srv,
		),
	)

	var err error

	err = asynqServer.RegisterSubscriber(srv, testTask1, handleTask1)

	err = asynqServer.RegisterSubscriber(srv, testDelayTask, handleDelayTask)

	err = asynqServer.RegisterSubscriber(srv, testPeriodicTask, handlePeriodicTask)

	// 最多重试3次，10秒超时，20秒后过期
	err = srv.NewTask(
		testTask1,
		&TaskPayload{Message: "delay task"},
		asynq.MaxRetry(3),
		asynq.Timeout(10*time.Second),
		asynq.Deadline(time.Now().Add(20*time.Second)),
		asynq.TaskID(testTask1),
	)

	// 延迟任务
	err = srv.NewTask(
		testDelayTask,
		&TaskPayload{Message: "delay task"},
		asynq.ProcessIn(3*time.Second),
		asynq.TaskID(testDelayTask),
	)

	// 周期性任务，每分钟执行一次
	_, err = srv.NewPeriodicTask(
		"*/1 * * * ?",
		testPeriodicTask,
		&TaskPayload{Message: "periodic task 1"},
		asynq.TaskID(testPeriodicTask+"1"),
	)

	_, err = srv.NewPeriodicTask(
		"*/1 * * * ?",
		testPeriodicTask,
		&TaskPayload{Message: "periodic task 2"},
		asynq.TaskID(testPeriodicTask+"2"),
	)

	_, err = srv.NewPeriodicTask(
		"*/1 * * * ?",
		testPeriodicTask,
		&TaskPayload{Message: "periodic task 3"},
		asynq.TaskID(testPeriodicTask+"3"),
	)

	//_, err = srv.NewPeriodicTask(
	//	"*/1 * * * ?",
	//	"test_periodic_",
	//	&TaskPayload{Message: "periodic task xxx"},
	//)

	if err = app.Run(); err != nil {
		log.Error(err)
	}

	<-interrupt
}
