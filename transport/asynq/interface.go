package asynq

import (
	"context"

	"github.com/hibiken/asynq"
	"github.com/tx7do/kratos-transport/broker"
)

// TaskScheduler 定义任务调度接口
type TaskScheduler interface {
	TaskTypeExists(taskType string) bool
	GetRegisteredTaskTypes() []string

	RegisterSubscriber(taskType string, handler MessageHandler, creator Creator) error
	RegisterSubscriberWithCtx(
		taskType string,
		handler func(context.Context, string, MessagePayload) error,
		creator Creator,
	) error

	NewTask(typeName string, msg broker.Any, opts ...asynq.Option) error
	NewWaitResultTask(typeName string, msg broker.Any, opts ...asynq.Option) error
	NewPeriodicTask(cronSpec, taskId, typeName string, msg broker.Any, opts ...asynq.Option) (string, error)

	RemovePeriodicTask(id string) error
	RemoveAllPeriodicTask()
}
