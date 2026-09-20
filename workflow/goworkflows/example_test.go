package goworkflows_test

import (
	"context"

	"github.com/cschleiden/go-workflows/backend/sqlite"
	"github.com/cschleiden/go-workflows/workflow"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/workflow/goworkflows"
)

// demoWorkflow 是示例用的工作流：收到名字后原样返回问候语。
func demoWorkflow(ctx workflow.Context, name string) (string, error) {
	return "hello, " + name, nil
}

// ExampleNewClient 演示基于内存 SQLite 后端创建工作流客户端与 Worker：
// 注册工作流、启动 Worker 并创建一次工作流实例。
// 没有 Output 注释，go test 只编译不执行；内存后端无需外部服务即可实际运行。
func ExampleNewClient() {
	ctx := context.Background()

	// 内存 SQLite 后端，也可以替换为 mysql/postgres/redis 等后端
	backend := sqlite.NewInMemoryBackend()
	defer func() { _ = backend.Close() }()

	// 创建 Worker 并注册工作流
	w, err := goworkflows.NewWorker(backend, &goworkflows.WorkerOptions{
		MaxParallelWorkflowTasks: 4,
		ActivityPollers:          1,
	})
	if err != nil {
		log.Error(err)
		return
	}
	if err := w.RegisterWorkflow(demoWorkflow); err != nil {
		log.Error(err)
		return
	}
	if err := w.Start(ctx); err != nil {
		log.Error(err)
		return
	}
	defer w.Stop()

	// 创建客户端并发起一次工作流实例
	client, err := goworkflows.NewClient(backend)
	if err != nil {
		log.Error(err)
		return
	}
	defer func() { _ = client.Close() }()

	instance, err := client.CreateWorkflowInstance(ctx, goworkflows.CreateWorkflowOptions{
		InstanceID: "demo-instance-1",
	}, demoWorkflow, "kratos")
	if err != nil {
		log.Error(err)
		return
	}

	log.Infof("started workflow instance: %s (execution: %s)", instance.InstanceID, instance.ExecutionID)

	// 等待工作流实例完成，timeout 为 0 时使用默认的 20 秒
	if err := client.WaitForWorkflowInstance(ctx, instance, 0); err != nil {
		log.Error(err)
		return
	}
}
