package mcp_test

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	mcpServer "github.com/tx7do/kratos-transport/transport/mcp"
)

// ExampleNewServer 演示以 Streamable HTTP 模式启动一个 MCP 工具服务：
// 注册一个四则运算计算器工具，供 AI 客户端调用。
// 没有 Output 注释，go test 只编译不执行；实际运行会阻塞等待进程退出信号。
func ExampleNewServer() {
	srv := mcpServer.NewServer(
		mcpServer.WithServerName("Calculator Demo"),
		mcpServer.WithServerVersion("1.0.0"),
		mcpServer.WithMCPServerOptions(
			server.WithToolCapabilities(false),
			server.WithRecovery(),
		),
		mcpServer.WithMCPServeType(mcpServer.ServerTypeHTTP),
		mcpServer.WithMCPServeAddress(":8080"),
	)

	calculatorTool := mcp.NewTool("calculate",
		mcp.WithDescription("Perform basic arithmetic operations"),
		mcp.WithString("operation",
			mcp.Required(),
			mcp.Description("The operation to perform (add, subtract, multiply, divide)"),
			mcp.Enum("add", "subtract", "multiply", "divide"),
		),
		mcp.WithNumber("x", mcp.Required(), mcp.Description("First number")),
		mcp.WithNumber("y", mcp.Required(), mcp.Description("Second number")),
	)

	if err := srv.RegisterHandler(calculatorTool, handleCalculate); err != nil {
		log.Error(err)
		return
	}

	app := kratos.New(
		kratos.Name("mcp"),
		kratos.Server(
			srv,
		),
	)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}

func handleCalculate(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	op, err := request.RequireString("operation")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	x, err := request.RequireFloat("x")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	y, err := request.RequireFloat("y")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var result float64
	switch op {
	case "add":
		result = x + y
	case "subtract":
		result = x - y
	case "multiply":
		result = x * y
	case "divide":
		if y == 0 {
			return mcp.NewToolResultError("cannot divide by zero"), nil
		}
		result = x / y
	}

	return mcp.NewToolResultText(fmt.Sprintf("%.2f", result)), nil
}
