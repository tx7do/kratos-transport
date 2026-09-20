package tcp

import "os"

// itIntegration 是否运行集成测试（依赖本地 tcp server）
var itIntegration = os.Getenv("KRATOS_IT") != ""
