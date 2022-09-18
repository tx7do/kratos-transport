package machinery

import (
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"os"
	"os/signal"
	"syscall"
	"testing"
)

const (
	localRedisAddr = "127.0.0.1:6379"

	testTask1        = "test_task_1"
	testTaskDelay    = "test_task_delay"
	testPeriodicTask = "test_periodic_task"
	sumTask          = "sum_task"
)

func TestNewTask(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		WithRedisAddress([]string{localRedisAddr}, []string{localRedisAddr}),
		//WithYamlConfig("./testconfig.yml", false),
	)

	var err error

	var args = map[string]interface{}{}
	args["int64"] = 1
	err = srv.NewTask(sumTask, args)
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
	log.Infof("Task Type: [%s], Payload: [%s]", task.Type(), string(task.Payload()))
	return nil
}

func handleAdd(args ...int64) (int64, error) {
	sum := int64(0)
	for _, arg := range args {
		sum += arg
	}
	fmt.Printf("sum: %d\n", sum)
	return sum, nil
}

func handlePeriodicTask() error {
	fmt.Println("################ 执行周期任务PeriodicTask #################")
	return nil
}

func TestTaskProcess(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		WithRedisAddress([]string{localRedisAddr}, []string{localRedisAddr}),
	)

	var err error

	err = srv.HandleFunc(testTask1, handleTask1)
	err = srv.HandleFunc(testTask1, handleTask1)
	assert.Nil(t, err)
	err = srv.HandleFunc(testTaskDelay, handleDelayTask)
	assert.Nil(t, err)
	err = srv.HandleFunc(sumTask, handleAdd)
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
		WithRedisAddress([]string{localRedisAddr}, []string{localRedisAddr}),
	)

	var err error

	err = srv.HandleFunc(testTask1, handleTask1)
	assert.Nil(t, err)
	err = srv.HandleFunc(testTaskDelay, handleDelayTask)
	assert.Nil(t, err)
	err = srv.HandleFunc(sumTask, handleAdd)
	assert.Nil(t, err)

	var args = map[string]interface{}{}
	args["int64"] = 1
	err = srv.NewTask(sumTask, args)
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
		WithRedisAddress([]string{localRedisAddr}, []string{localRedisAddr}),
	)

	var err error

	err = srv.HandleFunc(testPeriodicTask, handlePeriodicTask)
	assert.Nil(t, err)

	var args = map[string]interface{}{}
	// 每分钟执行一次
	err = srv.NewPeriodicTask("*/1 * * * ?", testPeriodicTask, args)
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
