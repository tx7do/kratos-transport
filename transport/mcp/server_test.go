package mcp

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"testing"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/stretchr/testify/assert"
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
		WithMCPServeType(ServerTypeHTTP),
		WithMCPServeAddress(":8080"),
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
		t.Logf("Received calculation request: %#v", request)

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

func TestClient(t *testing.T) {
	ctx := context.Background()

	httpTransport, err := transport.NewStreamableHTTP("http://localhost:8080/mcp")
	assert.NoError(t, err)
	assert.NotNil(t, httpTransport)

	mcpClient := client.NewClient(
		httpTransport,
	)
	assert.NotNil(t, mcpClient)
	defer mcpClient.Close()

	err = mcpClient.Start(ctx)
	assert.NoError(t, err)

	// Initialize the MCP session
	initRequest := mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			Capabilities:    mcp.ClientCapabilities{},
			ClientInfo: mcp.Implementation{
				Name:    "calculate-http-client",
				Version: "1.0.0",
			},
		},
	}

	_, err = mcpClient.Initialize(ctx, initRequest)
	assert.NoError(t, err)

	// 调用计算工具
	result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "calculate",
			Arguments: map[string]any{
				"operation": "add",
				"x":         7,
				"y":         8,
			},
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Content)

	if len(result.Content) > 0 {
		textContent, ok := result.Content[0].(mcp.TextContent)
		assert.True(t, ok, "expected TextContent type")
		assert.Equal(t, "15.00", textContent.Text)
		t.Logf("计算结果: %s", textContent.Text)
	}

	t.Logf("完整结果: %#v", result)
}

func TestServer_RegisterHandlerWithJsonString(t *testing.T) {
	jsonStr := `{
  "name": "echo",
  "description": "Echoes the input string",
  "inputSchema": {
	"type": "object",
	"properties": {
	  "message": {
		"type": "string",
		"description": "The message to echo"
	  }
	},
	"required": ["message"]
  }
}`

	ctx := context.Background()

	srv := NewServer(
		WithServerName("Echo Demo"),
		WithServerVersion("1.0.0"),
		WithMCPServerOptions(
			server.WithToolCapabilities(false),
			server.WithRecovery(),
		),
		WithMCPServeType(ServerTypeHTTP),
		WithMCPServeAddress(":8081"),
	)

	err := srv.RegisterHandlerWithJsonString(jsonStr, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		msg, err := request.RequireString("message")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(msg), nil
	})
	assert.NoError(t, err)

	if err := srv.Start(ctx); err != nil {
		panic(err)
	}

	defer func() {
		if err := srv.Stop(ctx); err != nil {
			t.Errorf("expected nil got %v", err)
		}
	}()

	// The server is now running with the echo tool registered.
	// You can implement a client test similar to TestClient to call the echo tool.

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-interrupt
}

func TestServer_RegisterHandlerWithJsonSchema(t *testing.T) {
	jsonSchemaStr := `{
	"type": "object",
	"properties": {
	  "text": {
		"type": "string",
		"description": "The text to reverse"
	  }
	},
	"required": ["text"]
  }`

	ctx := context.Background()

	srv := NewServer(
		WithServerName("Reverse Demo"),
		WithServerVersion("1.0.0"),
		WithMCPServerOptions(
			server.WithToolCapabilities(false),
			server.WithRecovery(),
		),
		WithMCPServeType(ServerTypeHTTP),
		WithMCPServeAddress(":8082"),
	)

	err := srv.RegisterHandlerWithJsonSchema("reverse", "Reverses the input text", jsonSchemaStr, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		text, err := request.RequireString("text")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Reverse the text
		runes := []rune(text)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		reversed := string(runes)

		return mcp.NewToolResultText(reversed), nil
	})
	assert.NoError(t, err)

	if err := srv.Start(ctx); err != nil {
		panic(err)
	}

	defer func() {
		if err := srv.Stop(ctx); err != nil {
			t.Errorf("expected nil got %v", err)
		}
	}()

	// The server is now running with the reverse tool registered.
	// You can implement a client test similar to TestClient to call the reverse tool.

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-interrupt
}
