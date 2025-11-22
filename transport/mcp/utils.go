package mcp

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
)

// LoadToolFromJsonFile 从 JSON 文件加载并转换为 mcp.Tool
func LoadToolFromJsonFile(filePath string) (mcp.Tool, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return mcp.Tool{}, fmt.Errorf("读取文件失败：%v", err)
	}

	var tool mcp.Tool
	if err := json.Unmarshal(data, &tool); err != nil {
		return mcp.Tool{}, fmt.Errorf("JSON 反序列化失败：%v", err)
	}

	if err := ValidateToolInputSchema(tool.InputSchema); err != nil {
		return mcp.Tool{}, fmt.Errorf("InputSchema 非法：%v", err)
	}

	return tool, nil
}

// LoadToolFromJsonString 从 JSON 字符串加载并转换为 mcp.Tool
func LoadToolFromJsonString(jsonStr string) (mcp.Tool, error) {
	var tool mcp.Tool
	if err := json.Unmarshal([]byte(jsonStr), &tool); err != nil {
		return mcp.Tool{}, fmt.Errorf("JSON 反序列化失败：%v", err)
	}

	return tool, ValidateToolInputSchema(tool.InputSchema)
}

// ValidateToolInputSchema 校验 InputSchema 基础合法性（避免 JSON 写错导致的结构问题）
func ValidateToolInputSchema(schema mcp.ToolInputSchema) error {
	if schema.Type == "" {
		return fmt.Errorf("InputSchema.type 不能为空（必须是 object）")
	}
	if schema.Type != "object" {
		return fmt.Errorf("InputSchema.type 必须为 object（当前：%s）", schema.Type)
	}
	if schema.Properties == nil {
		return fmt.Errorf("InputSchema.properties 不能为空")
	}

	return nil
}

func toRawMessage(jsonStr string) json.RawMessage {
	return json.RawMessage(jsonStr)
}
