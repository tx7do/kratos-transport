package mcp

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestServer(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		WithServerName("Calculator Demo"),
		WithServerVersion("1.0.0"),
		WithMCPServerOptions(
			server.WithToolCapabilities(false),
			server.WithRecovery(),
		),
	)

	calculatorTool := mcp.NewTool("calculate",
		mcp.WithDescription("Perform basic arithmetic operations"),
		mcp.WithString("operation",
			mcp.Required(),
			mcp.Description("The operation to perform (add, subtract, multiply, divide)"),
			mcp.Enum("add", "subtract", "multiply", "divide"),
		),
		mcp.WithNumber("x",
			mcp.Required(),
			mcp.Description("First number"),
		),
		mcp.WithNumber("y",
			mcp.Required(),
			mcp.Description("Second number"),
		),
	)

	_ = srv.RegisterHandler(calculatorTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Using helper functions for type-safe argument access
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
	})

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
