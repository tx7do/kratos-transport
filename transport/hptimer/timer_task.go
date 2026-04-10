package hptimer

import (
	"context"
	"time"
)

// Priority 事件优先级
type Priority int

const (
	PriorityHigh Priority = iota
	PriorityMedium
	PriorityLow
)

// TimerTaskID 定时任务唯一标识
type TimerTaskID string

// TimerTask 高精度定时任务结构体
// - 支持单次触发（At）或基于 Interval 的循环（Interval > 0）
// - 支持 Cron 表达式（Cron 非空）用于计算下次触发时间
// - Data 可携带任意负载（提交到事件调度器时会作为事件数据）
// - Callback 为任务执行函数；Ctx / cancel 用于取消任务
type TimerTask struct {
	ID       TimerTaskID                     // 任务ID
	At       time.Time                       // 首次/下次触发时间
	Interval time.Duration                   // 循环间隔（0 表示单次任务）
	Cron     string                          // 可选 cron 表达式（与 Interval 二选一）
	Data     any                             // 任务负载（提交到事件时使用）
	Priority Priority                        // 事件优先级
	Callback func(ctx context.Context) error // 任务回调
	Ctx      context.Context                 // 上下文（用于取消任务）
	cancel   context.CancelFunc              // 取消函数
}

// TimerTaskOption 可选参数类型
type TimerTaskOption func(*TimerTask)

// WithInterval 设置循环间隔
func WithInterval(d time.Duration) TimerTaskOption {
	return func(t *TimerTask) {
		t.Interval = d
	}
}

// WithCron 设置 cron 表达式
func WithCron(expr string) TimerTaskOption {
	return func(t *TimerTask) {
		t.Cron = expr
	}
}

// WithData 设置任务负载
func WithData(v any) TimerTaskOption {
	return func(t *TimerTask) {
		t.Data = v
	}
}

// WithCallback 设置任务回调
func WithCallback(cb func(ctx context.Context) error) TimerTaskOption {
	return func(t *TimerTask) {
		t.Callback = cb
	}
}

// WithPriority 设置事件优先级
func WithPriority(p Priority) TimerTaskOption {
	return func(t *TimerTask) {
		t.Priority = p
	}
}

// WithContext 使用提供的父上下文并为任务创建可取消子上下文
func WithContext(parent context.Context) TimerTaskOption {
	return func(t *TimerTask) {
		if parent == nil {
			return
		}
		t.Ctx, t.cancel = context.WithCancel(parent)
	}
}

// NewTimerTask 创建 TimerTask，至少需要 ID 与首次触发时间（At）
// 可通过可选参数配置 Interval/Cron/Data/Callback/Priority/Context
func NewTimerTask(id TimerTaskID, at time.Time, opts ...TimerTaskOption) *TimerTask {
	// 默认上下文与 cancel，方便外部直接调用 RemoveTask/Stop 时使用 cancel
	ctx, cancel := context.WithCancel(context.Background())

	t := &TimerTask{
		ID:       id,
		At:       at,
		Interval: 0,
		Cron:     "",
		Data:     nil,
		Callback: nil,
		Ctx:      ctx,
		cancel:   cancel,
		Priority: PriorityLow,
	}

	for _, o := range opts {
		o(t)
	}

	// 若选项中通过 WithContext 设置了 Ctx，则 WithContext 已创建 cancel；
	// 若外部通过某个选项直接替换了 Ctx 但没有设置 cancel，则确保为其创建 cancel。
	if t.Ctx != nil && t.cancel == nil {
		t.Ctx, t.cancel = context.WithCancel(t.Ctx)
	}

	return t
}
