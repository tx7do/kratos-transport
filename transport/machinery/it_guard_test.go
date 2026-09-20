package machinery

import (
	"fmt"
	"os"
	"testing"
)

// itIntegration 是否运行手工/集成测试：
// 这些测试依赖本地基础设施（Redis/Kafka/RabbitMQ/MQTT broker 等），
// 或以等待手动中断信号的方式长期运行，默认跳过。
var itIntegration = os.Getenv("KRATOS_IT") != ""

func TestMain(m *testing.M) {
	if !itIntegration {
		fmt.Fprintln(os.Stderr, "integration tests skipped (set KRATOS_IT=1 to enable)")
		os.Exit(0)
	}
	os.Exit(m.Run())
}
