package websocket

import (
	"fmt"
	"os"
	"testing"
)

// TestMain: 集成测试依赖本地 websocket server，默认跳过。
func TestMain(m *testing.M) {
	if os.Getenv("KRATOS_IT") == "" {
		fmt.Fprintln(os.Stderr, "integration tests skipped (set KRATOS_IT=1 to enable)")
		os.Exit(0)
	}
	os.Exit(m.Run())
}
