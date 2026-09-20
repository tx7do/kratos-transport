package temporal_test

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/tracing"
	"github.com/tx7do/kratos-transport/workflow/temporal"
)

// ExampleNewClient 演示连接 Temporal Server，开启链路追踪，
// 启动一个 Worker 监听任务队列，并发起一次工作流执行。
// 没有 Output 注释，go test 只编译不执行；实际运行需要本地 Temporal Server (localhost:7233)。
func ExampleNewClient() {
	ctx := context.Background()

	wc, err := temporal.NewClient(
		temporal.WithClientHostPort("localhost:7233"),
		temporal.WithClientNamespace("default"),
	)
	if err != nil {
		log.Error(err)
		return
	}
	defer wc.Close()

	// 开启生产者/消费者链路追踪，trace 经 OTLP gRPC 上报到 localhost:4317
	wc.WithTracing(tracing.WithTracerProvider(
		tracing.NewTracerProvider("otlp-grpc", "localhost:4317", "temporal_demo", "", "1.0.0", 1.0),
	))

	// 创建 Worker：自动注册默认的 BrokerMessageWorkflow，再挂上自定义 Activity
	w, err := wc.NewWorker(temporal.WorkerOptions{
		TaskQueue: "demo-task-queue",
		Activities: []any{
			func(ctx context.Context, body []byte) error {
				log.Infof("processing message: %s", string(body))
				return nil
			},
		},
	})
	if err != nil {
		log.Error(err)
		return
	}
	if err := w.Start(); err != nil {
		log.Error(err)
		return
	}
	defer w.Stop()

	// 发起一次异步工作流执行：未指定 WorkflowFn 时默认使用 BrokerMessageWorkflow
	runID, err := wc.Execute(ctx, []byte("hello temporal"), temporal.ExecuteOptions{
		TaskQueue:  "demo-task-queue",
		WorkflowID: "demo-workflow-1",
		RunTimeout: time.Minute,
	})
	if err != nil {
		log.Error(err)
		return
	}

	log.Infof("started workflow run: %s", runID)
}
