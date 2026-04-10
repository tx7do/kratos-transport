# kratos-transport 高精度定时任务引擎 — hptimer

基于 Kratos 传输层规范的高精度定时任务服务端，支持毫秒级定时、批量任务、动态增删、极低资源消耗，适合高频/高并发/大规模定时场景。

## 简介

`hptimer` 是 kratos-transport 生态中的高精度定时任务扩展，采用最小堆+单一 time.Timer 实现，任务触发延迟极低，资源消耗极小。可无缝集成 kratos 应用生命周期，支持自定义回调、优雅关闭。

## 特性

- 毫秒级定时精度，适合高频/密集任务
- 支持绝对时间、间隔、cron 表达式
- 任务批量管理，动态增删高效
- 资源消耗极低，单一 goroutine+timer
- 并发安全，支持高并发 Add/Remove
- 支持优雅关闭，任务执行完毕后退出
- 可自定义 TimerObserver，灵活集成业务
- 完全兼容 kratos transport.Server 生命周期

## 安装

```bash
go get github.com/tx7do/kratos-transport/transport/hptimer
```

## 快速使用

```go
import "github.com/tx7do/kratos-transport/transport/hptimer"

srv := hptimer.NewServer()

srv.AddTask(&hptimer.TimerTask{
    ID:       "task1",
    At:       time.Now().Add(100 * time.Millisecond),
    Callback: func(ctx context.Context) error {
        fmt.Println("task1 triggered")
        return nil
    },
})

srv.Start(context.Background())
// ...
srv.Stop(context.Background())
```

## 基准测试结果

在 `Intel(R) Core(TM) i7-14700HX` / `96G` / `Go 1.25` 环境下，`hptimer` 的基准测试结果如下：

| Benchmark                                 | N    | ns/op      | 说明                 |
|--------------------------------------------|------|------------|----------------------|
| BenchmarkHighPrecisionTimer_SingleTask-28  |  760 | 1,579,091  | 单任务添加+触发      |
| BenchmarkHighPrecisionTimer_BatchTasks-28  |  735 | 1,643,064  | 1000任务批量添加触发 |

- 单任务和批量 1000 任务的调度和触发，平均每轮耗时都在 1.5~1.6 ms 左右。
- 包含任务添加、调度、触发、回调等全流程，调度效率极高。

## 与 cron 对比

| 维度         | hptimer（高精度）         | cron（robfig/cron）         |
|--------------|--------------------------|-----------------------------|
| 精度         | 毫秒级                   | 秒级                        |
| 高频任务     | 高效                     | 不适合                      |
| 任务数量     | 性能几乎不变             | 任务多时调度压力增大        |
| 资源占用     | 1个timer+1个goroutine    | 任务多时资源线性增长        |
| 动态变更     | 高并发Add/Remove高效     | 频繁变更有锁竞争            |
| 表达能力     | 绝对/间隔/cron           | cron表达式                  |
| 适用场景     | 高精度/高频/批量/动态    | 周期性/低频/业务型          |

## 适用场景

- 高频/高精度定时任务（如毫秒级心跳、批量调度）
- 大量动态定时任务（如大规模定时推送、批量超时管理）
- 需要极低资源消耗的定时服务
- 替代传统 cron，提升性能和精度

## 贡献与反馈

欢迎 issue、PR 及建议！
