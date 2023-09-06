package asynq

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
)

const (
	localRedisAddr = "127.0.0.1:6379"

	testTask1        = "test_task_1"
	testDelayTask    = "test_delay_task"
	testPeriodicTask = "test_periodic_task"
)

type TaskPayload struct {
	Message string `json:"message"`
}

func handleTask1(taskType string, taskData *TaskPayload) error {
	LogInfof("Task Type: [%s], Payload: [%s]", taskType, taskData.Message)
	return nil
}

func handleDelayTask(taskType string, taskData *TaskPayload) error {
	LogInfof("Delay Task Type: [%s], Payload: [%s]", taskType, taskData.Message)
	return nil
}

func handlePeriodicTask(taskType string, taskData *TaskPayload) error {
	LogInfof("Periodic Task Type: [%s], Payload: [%s]", taskType, taskData.Message)
	return nil
}

func TestNewTaskOnly(t *testing.T) {
	//interrupt := make(chan os.Signal, 1)
	//signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	var err error

	srv := NewServer(
		WithAddress(localRedisAddr),
		WithRedisPassword("123456"),
	)

	err = srv.NewTask(testTask1,
		&TaskPayload{Message: "delay task"},
		asynq.MaxRetry(10),
		asynq.Timeout(3*time.Minute),
		asynq.ProcessIn(3*time.Second),
	)
	assert.Nil(t, err)

	if err = srv.Start(ctx); err != nil {
		panic(err)
	}

	defer func() {
		if err = srv.Stop(ctx); err != nil {
			t.Errorf("expected nil got %v", err)
		}
	}()

	//<-interrupt
}

func TestNewPeriodicTaskOnly(t *testing.T) {
	//interrupt := make(chan os.Signal, 1)
	//signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	var err error

	srv := NewServer(
		WithAddress(localRedisAddr),
		WithRedisPassword("123456"),
	)

	// 每分钟执行一次
	_, err = srv.NewPeriodicTask(
		"*/1 * * * ?",
		testPeriodicTask,
		&TaskPayload{Message: "periodic task"},
	)
	assert.Nil(t, err)

	if err = srv.Start(ctx); err != nil {
		panic(err)
	}

	defer func() {
		if err = srv.Stop(ctx); err != nil {
			t.Errorf("expected nil got %v", err)
		}
	}()

	//<-interrupt
}

func TestDelayTask(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()
	var err error

	srv := NewServer(
		WithAddress(localRedisAddr),
	)

	err = RegisterSubscriber[TaskPayload](srv,
		testDelayTask,
		handleDelayTask,
	)
	assert.Nil(t, err)

	// 延迟队列
	err = srv.NewTask(testDelayTask,
		&TaskPayload{Message: "delay task"},
		asynq.ProcessIn(3*time.Second),
	)
	assert.Nil(t, err)

	if err = srv.Start(ctx); err != nil {
		panic(err)
	}

	defer func() {
		if err = srv.Stop(ctx); err != nil {
			t.Errorf("expected nil got %v", err)
		}
	}()

	<-interrupt
}

func TestPeriodicTask(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()
	var err error

	srv := NewServer(
		WithAddress(localRedisAddr),
		WithRedisPassword("123456"),
	)

	err = RegisterSubscriber(srv,
		testPeriodicTask,
		handlePeriodicTask,
	)
	assert.Nil(t, err)

	// 每分钟执行一次
	_, err = srv.NewPeriodicTask(
		"*/1 * * * ?",
		testPeriodicTask,
		&TaskPayload{Message: "periodic task"},
	)
	assert.Nil(t, err)

	if err = srv.Start(ctx); err != nil {
		panic(err)
	}

	defer func() {
		if err = srv.Stop(ctx); err != nil {
			t.Errorf("expected nil got %v", err)
		}
	}()

	<-interrupt
}

func TestTaskSubscribe(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()
	var err error

	srv := NewServer(
		WithAddress(localRedisAddr),
		WithRedisPassword("123456"),
	)

	err = RegisterSubscriber[TaskPayload](srv,
		testTask1,
		handleTask1,
	)
	assert.Nil(t, err)

	err = RegisterSubscriber[TaskPayload](srv,
		testDelayTask,
		handleDelayTask,
	)
	assert.Nil(t, err)

	err = RegisterSubscriber[TaskPayload](srv,
		testPeriodicTask,
		handlePeriodicTask,
	)
	assert.Nil(t, err)

	if err = srv.Start(ctx); err != nil {
		panic(err)
	}

	defer func() {
		if err = srv.Stop(ctx); err != nil {
			t.Errorf("expected nil got %v", err)
		}
	}()

	<-interrupt
}

func TestAllInOne(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	var err error

	srv := NewServer(
		WithAddress(localRedisAddr),
	)

	err = RegisterSubscriber[TaskPayload](srv,
		testTask1,
		handleTask1,
	)
	assert.Nil(t, err)

	err = RegisterSubscriber[TaskPayload](srv,
		testDelayTask,
		handleDelayTask,
	)
	assert.Nil(t, err)

	err = RegisterSubscriber[TaskPayload](srv,
		testPeriodicTask,
		handlePeriodicTask,
	)
	assert.Nil(t, err)

	// 最多重试3次，10秒超时，20秒后过期
	err = srv.NewTask(testTask1,
		&TaskPayload{Message: "delay task"},
		asynq.MaxRetry(3),
		asynq.Timeout(10*time.Second),
		asynq.Deadline(time.Now().Add(20*time.Second)),
	)
	assert.Nil(t, err)

	// 延迟任务
	err = srv.NewTask(testDelayTask,
		&TaskPayload{Message: "delay task"},
		asynq.ProcessIn(3*time.Second),
	)
	assert.Nil(t, err)

	// 周期性任务，每分钟执行一次
	_, err = srv.NewPeriodicTask(
		"*/1 * * * ?",
		testPeriodicTask,
		&TaskPayload{Message: "periodic task"},
	)
	assert.Nil(t, err)

	if err = srv.Start(ctx); err != nil {
		panic(err)
	}

	defer func() {
		if err = srv.Stop(ctx); err != nil {
			t.Errorf("expected nil got %v", err)
		}
	}()

	<-interrupt
}
