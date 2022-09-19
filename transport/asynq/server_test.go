package asynq

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
)

const (
	localRedisAddr = "127.0.0.1:6379"

	testTask1        = "test_task_1"
	testDelayTask    = "test_delay_task"
	testPeriodicTask = "test_periodic_task"
)

func TestNewTask(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		WithAddress(localRedisAddr),
	)

	err := srv.NewTask(testTask1, []byte("test string"), asynq.MaxRetry(10), asynq.Timeout(3*time.Minute))
	assert.Nil(t, err)

	if err := srv.Start(ctx); err != nil {
		panic(err)
	}

	defer func() {
		if err := srv.Stop(ctx); err != nil {
			t.Errorf("expected nil got %v", err)
		}
	}()

	<-interrupt
}

func handleTask1(_ context.Context, task *asynq.Task) error {
	log.Infof("Task Type: [%s], Payload: [%s]", task.Type(), string(task.Payload()))
	return nil
}

func handleDelayTask(_ context.Context, task *asynq.Task) error {
	log.Infof("Delay Task Type: [%s], Payload: [%s]", task.Type(), string(task.Payload()))
	return nil
}

func handlePeriodicTask(_ context.Context, task *asynq.Task) error {
	log.Infof("Periodic Task Type: [%s], Payload: [%s]", task.Type(), string(task.Payload()))
	return nil
}

func TestTaskProcess(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		WithAddress(localRedisAddr),
	)

	err := srv.HandleFunc(testTask1, handleTask1)
	assert.Nil(t, err)
	err = srv.HandleFunc(testDelayTask, handleDelayTask)
	assert.Nil(t, err)

	if err := srv.Start(ctx); err != nil {
		panic(err)
	}

	defer func() {
		if err := srv.Stop(ctx); err != nil {
			t.Errorf("expected nil got %v", err)
		}
	}()

	<-interrupt
}

func TestAllInOne(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		WithAddress(localRedisAddr),
	)

	err := srv.HandleFunc(testTask1, handleTask1)
	assert.Nil(t, err)
	err = srv.HandleFunc(testDelayTask, handleDelayTask)
	assert.Nil(t, err)
	err = srv.HandleFunc(testPeriodicTask, handlePeriodicTask)
	assert.Nil(t, err)

	// 最多重试3次，10秒超时，20秒后过期
	err = srv.NewTask(testTask1, []byte("test string"),
		asynq.MaxRetry(10),
		asynq.Timeout(10*time.Second),
		asynq.Deadline(time.Now().Add(20*time.Second)))
	assert.Nil(t, err)

	// 延迟任务
	err = srv.NewTask(testDelayTask, []byte("delay task"), asynq.ProcessIn(3*time.Second))
	assert.Nil(t, err)

	// 周期性任务，每分钟执行一次
	err = srv.NewPeriodicTask("*/1 * * * ?", testPeriodicTask, []byte("periodic task"))
	assert.Nil(t, err)

	if err := srv.Start(ctx); err != nil {
		panic(err)
	}

	defer func() {
		if err := srv.Stop(ctx); err != nil {
			t.Errorf("expected nil got %v", err)
		}
	}()

	<-interrupt
}

func TestDelayTask(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		WithAddress(localRedisAddr),
	)

	var err error

	err = srv.HandleFunc(testDelayTask, handleDelayTask)
	assert.Nil(t, err)

	// 延迟队列
	err = srv.NewTask(testDelayTask, []byte("delay task"), asynq.ProcessIn(3*time.Second))
	assert.Nil(t, err)

	if err := srv.Start(ctx); err != nil {
		panic(err)
	}

	defer func() {
		if err := srv.Stop(ctx); err != nil {
			t.Errorf("expected nil got %v", err)
		}
	}()

	<-interrupt
}

func TestPeriodicTask(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		WithAddress(localRedisAddr),
	)

	var err error

	err = srv.HandleFunc(testPeriodicTask, handlePeriodicTask)
	assert.Nil(t, err)

	// 每分钟执行一次
	err = srv.NewPeriodicTask("*/1 * * * ?", testPeriodicTask, []byte("periodic task"))
	assert.Nil(t, err)

	if err := srv.Start(ctx); err != nil {
		panic(err)
	}

	defer func() {
		if err := srv.Stop(ctx); err != nil {
			t.Errorf("expected nil got %v", err)
		}
	}()

	<-interrupt
}
