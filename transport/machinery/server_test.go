package machinery

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/RichardKnop/machinery/v2/log"
	"github.com/stretchr/testify/assert"
)

const (
	localRedisAddr = "127.0.0.1:6379"

	testTask1        = "test_task_1"
	testDelayTask    = "test_delay_task"
	testPeriodicTask = "test_periodic_task"

	addTask         = "add"
	multiplyTask    = "multiply"
	sumIntTask      = "sum_ints"
	sumFloatTask    = "sum_floats"
	concatTask      = "concat"
	splitTask       = "split"
	panicTask       = "panic_task"
	longRunningTask = "long_running_task"
)

func handleTask1() error {
	fmt.Println("################ 执行任务Task1 #################")
	return nil
}

func handleDelayTask() error {
	fmt.Println("################ 执行延迟任务DelayTask #################")
	return nil
}

func handlePeriodicTask() error {
	fmt.Println("################ 执行周期任务PeriodicTask #################")
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

func handleMultiply(args ...int64) (int64, error) {
	sum := int64(1)
	for _, arg := range args {
		sum *= arg
	}
	fmt.Printf("mulptiply: %d\n", sum)
	return sum, nil
}

func handleSumInts(numbers []int64) (int64, error) {
	var sum int64
	for _, num := range numbers {
		sum += num
	}
	fmt.Printf("sum int: %d\n", sum)
	return sum, nil
}

func handleSumFloats(numbers []float64) (float64, error) {
	var sum float64
	for _, num := range numbers {
		sum += num
	}
	fmt.Printf("sum float: %f\n", sum)
	return sum, nil
}

func handleConcat(strs []string) (string, error) {
	var res string
	for _, s := range strs {
		res += s
	}
	fmt.Printf("concat string: %s\n", res)
	return res, nil
}

func handleSplit(str string) ([]string, error) {
	res := strings.Split(str, "")
	fmt.Printf("split string: %s\n", res)
	return res, nil
}

func handlePanicTask() (string, error) {
	panic(errors.New("oops"))
}

func handleLongRunningTask() error {
	log.INFO.Print("Long running task started")
	for i := 0; i < 10; i++ {
		log.INFO.Print(10 - i)
		time.Sleep(1 * time.Second)
	}
	log.INFO.Print("Long running task finished")
	return nil
}

func registerHandler(t *testing.T, srv *Server) {
	err := srv.HandleFunc(addTask, handleAdd)
	assert.Nil(t, err)
	err = srv.HandleFunc(multiplyTask, handleMultiply)
	assert.Nil(t, err)
	err = srv.HandleFunc(sumIntTask, handleSumInts)
	assert.Nil(t, err)
	err = srv.HandleFunc(sumFloatTask, handleSumFloats)
	assert.Nil(t, err)
	err = srv.HandleFunc(concatTask, handleConcat)
	assert.Nil(t, err)
	err = srv.HandleFunc(splitTask, handleSplit)
	assert.Nil(t, err)
	err = srv.HandleFunc(panicTask, handlePanicTask)
	assert.Nil(t, err)
	err = srv.HandleFunc(longRunningTask, handleLongRunningTask)
	assert.Nil(t, err)
}

func TestNewTask(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		WithRedisAddress([]string{localRedisAddr}, []string{localRedisAddr}),
		//WithYamlConfig("./testconfig.yml", false),
	)

	var err error

	err = srv.HandleFunc(testTask1, handleTask1)
	assert.Nil(t, err)
	err = srv.HandleFunc(addTask, handleAdd)
	assert.Nil(t, err)

	err = srv.NewTask(testTask1)
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

func TestTaskProcess(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		WithRedisAddress([]string{localRedisAddr}, []string{localRedisAddr}),
	)

	var err error

	err = srv.HandleFunc(testTask1, handleTask1)
	assert.Nil(t, err)
	err = srv.HandleFunc(testDelayTask, handleDelayTask)
	assert.Nil(t, err)
	err = srv.HandleFunc(testPeriodicTask, handlePeriodicTask)
	assert.Nil(t, err)
	err = srv.HandleFunc(addTask, handleAdd)
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
	err = srv.HandleFunc(testDelayTask, handleDelayTask)
	assert.Nil(t, err)
	err = srv.HandleFunc(testPeriodicTask, handlePeriodicTask)
	assert.Nil(t, err)
	err = srv.HandleFunc(addTask, handleAdd)
	assert.Nil(t, err)

	err = srv.NewTask(addTask, WithArgument("int64", 1))
	assert.Nil(t, err)

	// 每分钟执行一次
	err = srv.NewPeriodicTask("*/1 * * * ?", testPeriodicTask)
	assert.Nil(t, err)

	// 延迟5秒执行任务
	err = srv.NewTask(testDelayTask, WithDelayTime(time.Now().UTC().Add(time.Second*5)))
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
		WithRedisAddress([]string{localRedisAddr}, []string{localRedisAddr}),
	)

	var err error

	err = srv.HandleFunc(testDelayTask, handleDelayTask)
	assert.Nil(t, err)

	// 延迟5秒执行任务
	err = srv.NewTask(testDelayTask, WithDelayTime(time.Now().UTC().Add(time.Second*5)))
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

	// 每分钟执行一次
	err = srv.NewPeriodicTask("*/1 * * * ?", testPeriodicTask)
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

func TestWorkflows_Groups(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		WithRedisAddress([]string{localRedisAddr}, []string{localRedisAddr}),
	)

	var err error

	registerHandler(t, srv)

	err = srv.NewGroup(
		WithTask(addTask, WithArgument("int64", 1), WithArgument("int64", 1)),
		WithTask(addTask, WithArgument("int64", 5), WithArgument("int64", 5)),
	)
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

func TestWorkflows_Chords(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		WithRedisAddress([]string{localRedisAddr}, []string{localRedisAddr}),
	)

	var err error

	registerHandler(t, srv)

	// multiply(add(1, 1), add(5, 5))
	// (1 + 1) * (5 + 5) = 2 * 10 = 20

	err = srv.NewChord(
		WithTask(addTask, WithArgument("int64", 1), WithArgument("int64", 1)),
		WithTask(addTask, WithArgument("int64", 5), WithArgument("int64", 5)),
		WithTask(multiplyTask),
	)
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

func TestWorkflows_Chains(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		WithRedisAddress([]string{localRedisAddr}, []string{localRedisAddr}),
	)

	var err error

	registerHandler(t, srv)

	// multiply(4, add(5, 5, add(1, 1)))
	//   4 * (5 + 5 + (1 + 1))   # task1: add(1, 1)        returns 2
	// = 4 * (5 + 5 + 2)         # task2: add(5, 5, 2)     returns 12
	// = 4 * (12)                # task3: multiply(4, 12)  returns 48
	// = 48

	err = srv.NewChain(
		WithTask(addTask, WithArgument("int64", 1), WithArgument("int64", 1)),
		WithTask(addTask, WithArgument("int64", 5), WithArgument("int64", 5)),
		WithTask(multiplyTask, WithArgument("int64", 4)),
	)
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
