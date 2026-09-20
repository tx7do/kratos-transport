package argo_test

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/workflow/argo"
)

// ExampleNewClient 演示连接 Argo Server，提交一个 hello-world 工作流并查询工作流列表。
// 没有 Output 注释，go test 只编译不执行；实际运行需要可访问的 Argo Server (https://localhost:2746)。
func ExampleNewClient() {
	ctx := context.Background()

	client, err := argo.NewClient(argo.ClientOptions{
		ServerURL:          "https://localhost:2746",
		Namespace:          "default",
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Error(err)
		return
	}
	defer client.Close()

	// 提交一个包含单个 whalesay 容器模板的工作流
	wf, err := client.SubmitWorkflow(ctx, &argo.Workflow{
		APIVersion: "argoproj.io/v1alpha1",
		Kind:       "Workflow",
		Metadata: argo.ObjectMeta{
			GenerateName: "hello-world-",
			Namespace:    "default",
		},
		Spec: argo.WorkflowSpec{
			Entrypoint: "whalesay",
			Templates: []argo.Template{
				{
					Name: "whalesay",
					Container: &argo.Container{
						Image:   "docker/whalesay:latest",
						Command: []string{"cowsay"},
						Args:    []string{"hello kratos-transport"},
					},
				},
			},
		},
	}, &argo.SubmitOptions{})
	if err != nil {
		log.Error(err)
		return
	}

	log.Infof("submitted workflow: %s", wf.Metadata.Name)

	// 按标签过滤并分页查询命名空间下的工作流
	list, err := client.ListWorkflows(ctx, &argo.ListOptions{
		Namespace:     "default",
		LabelSelector: "workflows.argoproj.io/archive-strategy=false",
		Limit:         10,
	})
	if err != nil {
		log.Error(err)
		return
	}

	log.Infof("found %d workflows", len(list.Items))
}
