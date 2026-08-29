package asynq

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
)

const (
	localRedisURI = "redis://:*Abcd123456@127.0.0.1:6379"

	testTask1        = "test_task_1"
	testDelayTask    = "test_delay_task"
	testPeriodicTask = "test_periodic_task"
)

type TaskPayload struct {
	Message string `json:"message"`
}

func handleTask1(taskType string, taskData *TaskPayload) error {
	LogInfof("[%s] Task Type: [%s], Payload: [%s]", time.Now().Format("2006-01-02 15:04:05"), taskType, taskData.Message)
	return nil
}

func handleDelayTask(taskType string, taskData *TaskPayload) error {
	LogInfof("[%s] Delay Task Type: [%s], Payload: [%s]", time.Now().Format("2006-01-02 15:04:05"), taskType, taskData.Message)
	return nil
}

func handlePeriodicTask(taskType string, taskData *TaskPayload) error {
	LogInfof("[%s] Periodic Task Type: [%s], Payload: [%s]", time.Now().Format("2006-01-02 15:04:05"), taskType, taskData.Message)
	return nil
}

func TestNewTaskOnly(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	var err error

	srv := NewServer(
		WithRedisURI(localRedisURI),
		WithShutdownTimeout(3*time.Second),
	)

	err = srv.NewTask(testTask1,
		&TaskPayload{Message: "delay task"},
		asynq.MaxRetry(10),
		asynq.Timeout(3*time.Minute),
		asynq.ProcessIn(3*time.Second),
	)
	assert.Nil(t, err)

	if err = srv.Start(t.Context()); err != nil {
		panic(err)
	}

	defer func() {
		if err = srv.Stop(t.Context()); err != nil {
			t.Errorf("expected nil got %v", err)
		}
	}()

	<-interrupt
}

func TestNewPeriodicTaskOnly(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	var err error

	srv := NewServer(
		WithRedisURI(localRedisURI),
		WithShutdownTimeout(3*time.Second),
	)

	// 每分钟执行一次
	_, err = srv.NewPeriodicTask(
		"*/1 * * * ?",
		testPeriodicTask,
		&TaskPayload{Message: "periodic task"},
		asynq.Unique(time.Second*10),
	)
	assert.Nil(t, err)

	if err = srv.Start(t.Context()); err != nil {
		panic(err)
	}

	defer func() {
		if err = srv.Stop(t.Context()); err != nil {
			t.Errorf("expected nil got %v", err)
		}
	}()

	<-interrupt
}

func TestDelayTask(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	var err error

	srv := NewServer(
		WithRedisURI(localRedisURI),
		WithShutdownTimeout(3*time.Second),
	)

	err = RegisterSubscriber(srv, testDelayTask, handleDelayTask)
	assert.Nil(t, err)

	// 延迟队列，推迟5秒执行
	err = srv.NewTask(testDelayTask,
		&TaskPayload{
			Message: fmt.Sprintf("ProcessIn:[%s]", time.Now().Format("2006/1/2 15:04:05")),
		},
		asynq.ProcessIn(5*time.Second),
	)
	assert.Nil(t, err)

	// 延迟队列，指定时间点，3分钟后执行。
	err = srv.NewTask(testDelayTask,
		&TaskPayload{
			Message: fmt.Sprintf("ProcessAt:[%s]", time.Now().Format("2006/1/2 15:04:05")),
		},
		asynq.ProcessAt(time.Now().Add(3*time.Minute)),
	)
	assert.Nil(t, err)

	if err = srv.Start(t.Context()); err != nil {
		panic(err)
	}

	defer func() {
		if err = srv.Stop(t.Context()); err != nil {
			t.Errorf("expected nil got %v", err)
		}
	}()

	<-interrupt
}

func TestPeriodicTask(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	var err error

	srv := NewServer(
		WithRedisURI(localRedisURI),
		WithShutdownTimeout(3*time.Second),
	)

	err = RegisterSubscriber(srv, testPeriodicTask, handlePeriodicTask)
	assert.Nil(t, err)

	// 每分钟执行一次
	_, err = srv.NewPeriodicTask(
		"*/1 * * * ?",
		testPeriodicTask,
		&TaskPayload{Message: "periodic task"},
	)
	assert.Nil(t, err)

	if err = srv.Start(t.Context()); err != nil {
		panic(err)
	}

	defer func() {
		if err = srv.Stop(t.Context()); err != nil {
			t.Errorf("expected nil got %v", err)
		}
	}()

	<-interrupt
}

func TestTaskSubscribe(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	var err error

	srv := NewServer(
		WithRedisURI(localRedisURI),
		WithShutdownTimeout(3*time.Second),
	)

	err = RegisterSubscriber(srv, testTask1, handleTask1)
	assert.Nil(t, err)

	err = RegisterSubscriber(srv, testDelayTask, handleDelayTask)
	assert.Nil(t, err)

	err = RegisterSubscriber(srv, testPeriodicTask, handlePeriodicTask)
	assert.Nil(t, err)

	if err = srv.Start(t.Context()); err != nil {
		panic(err)
	}

	defer func() {
		if err = srv.Stop(t.Context()); err != nil {
			t.Errorf("expected nil got %v", err)
		}
	}()

	<-interrupt
}

func TestAllInOne(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	var err error

	srv := NewServer(
		WithRedisURI(localRedisURI),
		WithShutdownTimeout(3*time.Second),
	)

	err = RegisterSubscriber(srv, testTask1, handleTask1)
	assert.Nil(t, err)

	err = RegisterSubscriber(srv, testDelayTask, handleDelayTask)
	assert.Nil(t, err)

	err = RegisterSubscriber(srv, testPeriodicTask, handlePeriodicTask)
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

	if err = srv.Start(t.Context()); err != nil {
		panic(err)
	}

	defer func() {
		if err = srv.Stop(t.Context()); err != nil {
			t.Errorf("expected nil got %v", err)
		}
	}()

	<-interrupt
}

func TestWaitResultTask(t *testing.T) {

	var err error

	srv := NewServer(
		WithRedisURI(localRedisURI),
		WithShutdownTimeout(3*time.Second),
	)

	err = RegisterSubscriber(srv, testTask1, handleTask1)

	defer func() {
		if err = srv.Stop(t.Context()); err != nil {
			t.Errorf("expected nil got %v", err)
		}
	}()

	go func() {
		if err = srv.Start(t.Context()); err != nil {
			panic(err)
		}
	}()

	// 最多重试3次，10秒超时
	err = srv.NewWaitResultTask(testTask1,
		&TaskPayload{Message: "wait result task"},
		asynq.Retention(time.Hour*1),
		asynq.MaxRetry(3),
		asynq.Timeout(10*time.Second),
	)
	if err != nil {
		t.Errorf("expected nil got %v", err)
		return
	}

	t.Logf("Wait for task result...")
}

// TestPeriodicTaskSameTypeName verifies that multiple periodic tasks sharing
// the same typeName are tracked and removable independently:
//   - RemovePeriodicTaskByID removes any single entry by the entryID returned
//     by NewPeriodicTask, even when its typeName was overwritten by a later
//     registration;
//   - RemovePeriodicTask (by typeName) removes the latest entry and actually
//     cleans the entryIDs map (regression for the no-op cleanup bug).
func TestPeriodicTaskSameTypeName(t *testing.T) {
	srv := NewServer()
	assert.Nil(t, srv.createAsynqScheduler())

	const typeName = "test_periodic_same_typename"
	task := asynq.NewTask(typeName, nil)

	// two periodic tasks sharing the same typeName: the underlying scheduler
	// holds two entries, while entryIDs keeps only the latest one.
	id1, err := srv.scheduler.Register("*/5 * * * *", task)
	assert.Nil(t, err)
	srv.addPeriodicTaskEntryID(typeName, id1)

	id2, err := srv.scheduler.Register("*/7 * * * *", task)
	assert.Nil(t, err)
	srv.addPeriodicTaskEntryID(typeName, id2)

	assert.NotEqual(t, id1, id2)
	assert.Equal(t, id2, srv.QueryPeriodicTaskEntryID(typeName))

	// the overwritten entry is still firing in the scheduler and can only be
	// removed via its entryID.
	assert.Nil(t, srv.RemovePeriodicTaskByID(id1))
	assert.NotNil(t, srv.scheduler.Unregister(id1)) // already unregistered
	assert.Equal(t, id2, srv.QueryPeriodicTaskEntryID(typeName))

	// removing by typeName works for the latest entry and cleans the map.
	assert.Nil(t, srv.RemovePeriodicTask(typeName))
	assert.Equal(t, "", srv.QueryPeriodicTaskEntryID(typeName))
	assert.NotNil(t, srv.scheduler.Unregister(id2)) // already unregistered

	// RemovePeriodicTaskByID also clears the map entry that holds the entryID.
	id3, err := srv.scheduler.Register("*/9 * * * *", task)
	assert.Nil(t, err)
	srv.addPeriodicTaskEntryID(typeName, id3)
	assert.Nil(t, srv.RemovePeriodicTaskByID(id3))
	assert.Equal(t, "", srv.QueryPeriodicTaskEntryID(typeName))

	// removing an unknown or empty entryID fails instead of silently succeeding.
	assert.NotNil(t, srv.RemovePeriodicTaskByID("nonexistent-entry-id"))
	assert.NotNil(t, srv.RemovePeriodicTaskByID(""))
}
