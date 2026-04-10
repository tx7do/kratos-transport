# kratos-transport 定时任务扩展 — cron

基于 Kratos 传输层规范封装的 cron 定时任务服务端，可无缝接入 kratos 应用生命周期，支持秒级 cron 表达式、任务动态管理、优雅关闭与服务发现心跳。

## 简介

`cron` 是 kratos-transport 生态中的定时任务扩展，遵循 kratos `transport.Server` 标准接口设计，让 cron 定时任务以 “服务” 的形式接入
Kratos 应用，统一由 app 管理启动、停止与优雅退出，避免手动维护 cron 实例生命周期。

## 特性

- 遵循 Kratos `transport.Server` 规范，无缝集成 kratos app 生命周期
- 支持秒级 cron 表达式（6 位 / 5 位通用格式）
- 任务动态管理：新增、移除、停止单个 / 全部任务
- 内置并发安全保护，支持多协程安全操作
- 优雅关闭：等待运行中任务执行完毕再退出
- 可集成 keepalive 实现服务注册与心跳上报
- 标准日志输出，便于监控与问题排查
- 轻量化、无侵入、可直接在现有项目使用

## 依赖

- Go 1.18+
- kratos v2
- robfig/cron/v3
- kratos-transport/keepalive

## 安装

```bash
go get github.com/tx7do/kratos-transport/transport/cron
```

## 核心结构

### Server

cron 服务主结构，实现以下接口：

- `kratosTransport.Server`：启动、停止
- `kratosTransport.Endpointer`：获取服务端点（用于服务注册）

### ServerOption

用于配置服务实例：

- `WithEnableKeepAlive`：启用 / 禁用心跳
- `WithGracefullyShutdown`：开启 / 关闭优雅退出

## 快速使用

### 1. 创建并启动 cron 服务

```go
package main

import (
    "context"
    "log"

    "github.com/go-kratos/kratos/v2"
    "github.com/tx7do/kratos-transport/transport/cron"
)

func main() {
    // 创建 cron 服务
    cronSrv := cron.NewServer(
        cron.WithKeepalive(true),
    )

    // 注册到 kratos app
    app := kratos.New(
        kratos.Name("cron-demo"),
        kratos.Server(cronSrv),
    )

    // app 启动后添加任务
    go func() {
        // 每 10 秒执行一次
        _, _ = cronSrv.StartTimerJob("*/10 * * * * *", func() {
            log.Println("task run every 10 seconds")
        })

        // 每分钟执行一次
        _, _ = cronSrv.StartTimerJob("0 */1 * * * *", func() {
            log.Println("task run every minute")
        })
    }()

    // 启动应用
    if err := app.Run(); err != nil {
        log.Fatal(err)
    }
}
```

### 2. 任务管理

```go
// 添加任务
entryID, err := cronSrv.StartTimerJob("0 0 12 * * *", func() {
    // 每天中午12点执行
})

// 停止单个任务
cronSrv.StopTimerJob(entryID)

// 停止所有任务
cronSrv.StopAllJobs()

// 获取当前任务数量
count := cronSrv.GetJobCount()
```

## Cron 表达式支持

```text
# 秒 分 时 日 月 周
*/10 * * * * *       → 每 10 秒
0 */1 * * * *        → 每分钟
0 0 12 * * *         → 每天 12:00
0 0 12 * * 1-5       → 工作日 12:00
```

## 生命周期说明

1. `Start(ctx)`：启动 cron 调度器与心跳服务
2. `Stop(ctx)`：优雅停止调度器，等待任务执行完毕
3. 应用退出时自动触发 Stop，保证任务不被强行中断

## 适用场景

- 后台定时数据统计、清理
- 定时同步、对账、消息推送
- 定时任务统一管理平台
- 微服务内部轻量定时任务
- 需要与 kratos 生命周期绑定的定时逻辑

## 优势

- 不依赖第三方调度中间件（无 etcd / 无数据库） 
- 接入成本极低，符合 kratos 编码风格 
- 可直接配合服务注册发现实现高可用 
- 适合单体、微服务通用场景
