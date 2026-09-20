package trpc

import (
	"fmt"
	"os"
	"testing"
)

// TestMain: 集成测试会阻塞等待手动中断信号，默认跳过。
func TestMain(m *testing.M) {
	if os.Getenv("KRATOS_IT") == "" {
		fmt.Fprintln(os.Stderr, "integration tests skipped (set KRATOS_IT=1 to enable)")
		os.Exit(0)
	}
	os.Exit(m.Run())
}
