package conductor

import (
	"fmt"
	"sync"
	"time"

	"github.com/conductor-sdk/conductor-go/sdk/model"
	"github.com/conductor-sdk/conductor-go/sdk/worker"
)

// TaskHandler is a function that processes a Conductor task.
// It receives a Task and returns the output or an error.
type TaskHandler = model.ExecuteTaskFunction

// TaskWorker manages a Conductor task worker that polls for and executes tasks.
type TaskWorker struct {
	mu         sync.RWMutex
	client     *WorkflowClient
	taskRunner *worker.TaskRunner
	config     WorkerConfig
	stopped    bool
}

// StartWorker starts a task worker on the given WorkflowClient.
// This is the simplest way to start processing tasks.
func (wc *WorkflowClient) StartWorker(taskType string, handler TaskHandler, concurrency int, pollInterval time.Duration) (*TaskWorker, error) {
	if wc.apiClient == nil {
		return nil, fmt.Errorf("api client is nil")
	}

	if concurrency <= 0 {
		concurrency = defaultConcurrency
	}
	if pollInterval <= 0 {
		pollInterval = defaultPollInterval
	}

	taskRunner := worker.NewTaskRunnerWithApiClient(wc.apiClient)

	tw := &TaskWorker{
		client:     wc,
		taskRunner: taskRunner,
		config: WorkerConfig{
			TaskType:     taskType,
			Concurrency:  concurrency,
			PollInterval: pollInterval,
		},
	}

	taskRunner.StartWorker(taskType, handler, concurrency, pollInterval)

	LogInfof("started worker for task type: %s (concurrency: %d)", taskType, concurrency)

	return tw, nil
}

// StartWorkerWithConfig starts a task worker with full configuration.
func (wc *WorkflowClient) StartWorkerWithConfig(config WorkerConfig, handler TaskHandler) (*TaskWorker, error) {
	if wc.apiClient == nil {
		return nil, fmt.Errorf("api client is nil")
	}

	if config.Concurrency <= 0 {
		config.Concurrency = defaultConcurrency
	}
	if config.PollInterval <= 0 {
		config.PollInterval = defaultPollInterval
	}

	taskRunner := worker.NewTaskRunnerWithApiClient(wc.apiClient)

	tw := &TaskWorker{
		client:     wc,
		taskRunner: taskRunner,
		config:     config,
	}

	taskRunner.StartWorker(config.TaskType, handler, config.Concurrency, config.PollInterval)

	LogInfof("started worker for task type: %s (concurrency: %d, poll: %s)",
		config.TaskType, config.Concurrency, config.PollInterval)

	return tw, nil
}

// Stop 停止任务 worker：调用 SDK 的 Shutdown 触发该任务类型的轮询 goroutine 收敛。
func (tw *TaskWorker) Stop() {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.stopped {
		return
	}

	// SDK 提供 Shutdown(taskName)（配合 WaitWorkers 优雅收敛），此前注释称无停止 API 为误
	tw.taskRunner.Shutdown(tw.config.TaskType)
	tw.stopped = true

	LogInfof("worker for task type %s stopped", tw.config.TaskType)
}

// TaskType returns the task type this worker is polling for.
func (tw *TaskWorker) TaskType() string {
	return tw.config.TaskType
}

// IsRunning returns whether the worker is still running.
func (tw *TaskWorker) IsRunning() bool {
	tw.mu.RLock()
	defer tw.mu.RUnlock()
	return !tw.stopped
}
