package conductor_test

import (
	"context"
	"time"

	"github.com/conductor-sdk/conductor-go/sdk/model"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/workflow/conductor"
)

// ExampleNewClient 演示连接 Conductor Server，启动一个任务 Worker，
// 并发起一次工作流执行。
// 没有 Output 注释，go test 只编译不执行；实际运行需要本地 Conductor Server (http://localhost:8080/api)。
func ExampleNewClient() {
	ctx := context.Background()

	client, err := conductor.NewClient(conductor.ClientOptions{
		ServerURL: "http://localhost:8080/api",
	})
	if err != nil {
		log.Error(err)
		return
	}
	defer client.Close()

	// 启动任务 Worker：轮询并处理 demo_task 类型的任务，并发 2、轮询间隔 1s
	w, err := client.StartWorker("demo_task", func(task *model.Task) (interface{}, error) {
		log.Infof("processing task: %s, input: %v", task.TaskType, task.InputData)
		return map[string]interface{}{"greeting": "hello from worker"}, nil
	}, 2, time.Second)
	if err != nil {
		log.Error(err)
		return
	}
	defer w.Stop()

	// 发起一次异步工作流执行，输入会传递给工作流中的任务
	workflowID, err := client.StartWorkflow(ctx, conductor.StartWorkflowOptions{
		Name:          "demo_workflow",
		Input:         map[string]interface{}{"name": "kratos"},
		CorrelationID: "demo-correlation-1",
	})
	if err != nil {
		log.Error(err)
		return
	}

	log.Infof("started workflow: %s", workflowID)
}
