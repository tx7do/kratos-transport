package hptimer

import (
	"container/heap"
	"context"
	"sync"
	"time"

	"github.com/gorhill/cronexpr"
)

// TimerObserver 高精度定时器观察者
// 外部实现该接口即可接收任务触发事件
type TimerObserver interface {
	OnTimerTrigger(task *TimerTask)
}

// HighPrecisionTimer 高精度定时器引擎
type HighPrecisionTimer struct {
	heap    timerHeap                  // 任务最小堆
	mu      sync.Mutex                 // 堆操作锁
	timer   *time.Timer                // 核心定时器（单次触发，动态重置）
	running bool                       // 引擎运行状态
	wg      sync.WaitGroup             // 退出等待组
	tasks   map[TimerTaskID]*TimerTask // 任务索引（快速查找/删除）

	// 停止上下文
	ctx    context.Context
	cancel context.CancelFunc

	wakeup chan struct{} // 唤醒通道

	observer   TimerObserver // 观察者（解耦）
	observerMu sync.RWMutex
}

// NewHighPrecisionTimer 创建高精度定时器引擎
func NewHighPrecisionTimer(observer TimerObserver) *HighPrecisionTimer {
	ctx, cancel := context.WithCancel(context.Background())

	ht := &HighPrecisionTimer{
		heap:     make(timerHeap, 0),
		tasks:    make(map[TimerTaskID]*TimerTask),
		observer: observer,
		running:  false,
		ctx:      ctx,
		cancel:   cancel,
		wakeup:   make(chan struct{}, 1),
	}

	if observer == nil {
		ht.observer = ht // 默认自己实现观察者接口（空实现）
	}

	heap.Init(&ht.heap)

	return ht
}

// SetObserver 设置观察者（外部订阅触发事件）
func (ht *HighPrecisionTimer) SetObserver(obs TimerObserver) {
	ht.observerMu.Lock()
	defer ht.observerMu.Unlock()
	ht.observer = obs
}

// Start 启动高精度定时器引擎
func (ht *HighPrecisionTimer) Start() {
	ht.mu.Lock()
	defer ht.mu.Unlock()

	if ht.running {
		LogWarnf("高精度定时器引擎已启动，无需重复调用")
		return
	}
	ht.running = true

	ht.wg.Add(1)
	go ht.run() // 启动主循环

	LogDebugf("高精度定时器引擎启动成功")
}

// Stop 停止定时器引擎（优雅退出）
func (ht *HighPrecisionTimer) Stop() {
	// 先取消上下文，通知run循环退出
	if ht.cancel != nil {
		ht.cancel()
	}

	ht.mu.Lock()
	if !ht.running {
		ht.mu.Unlock()
		return
	}
	ht.running = false

	// 停止定时器
	if ht.timer != nil {
		ht.timer.Stop()
	}

	// 取消所有任务
	for _, task := range ht.tasks {
		if task.cancel != nil {
			task.cancel()
		}
	}
	ht.mu.Unlock()

	// 等待主循环退出
	ht.wg.Wait()
}

// AddTask 添加定时任务
// 返回：任务ID（用于删除/修改）
func (ht *HighPrecisionTimer) AddTask(task *TimerTask) TimerTaskID {
	ht.mu.Lock()
	defer ht.mu.Unlock()

	// 校验入参
	if !ht.running || task == nil || task.ID == "" {
		LogWarnf("添加任务失败：引擎未运行/任务为空/ID为空，任务ID：%s", task.ID)
		return ""
	}

	// 初始化任务上下文
	if task.Ctx == nil {
		task.Ctx, task.cancel = context.WithCancel(context.Background())
	}

	// 初始化Cron任务的At时间
	if task.At.IsZero() && task.Cron != "" {
		if expr, err := cronexpr.Parse(task.Cron); err == nil {
			task.At = expr.Next(time.Now())
		} else {
			LogWarnf("解析Cron失败：%v，任务ID：%s", err, task.ID)
			return ""
		}
	}

	// 过滤At为零值的无效任务
	if task.At.IsZero() {
		LogWarnf("任务At时间为零值，任务ID：%s", task.ID)
		return ""
	}

	// 避免重复添加
	if _, exists := ht.tasks[task.ID]; exists {
		LogWarnf("任务已存在，任务ID：%s", task.ID)
		return ""
	}

	// 入堆并记录索引
	heap.Push(&ht.heap, task)
	ht.tasks[task.ID] = task

	// 如果新加入的任务成为堆顶（即比之前的最早任务更早），通知 run 重置 timer
	// 非阻塞发送，避免因信号未被消费而阻塞 AddTask
	if ht.timer != nil && ht.heap.Len() > 0 && ht.heap[0] == task {
		select {
		case ht.wakeup <- struct{}{}:
		default:
		}
	}

	return task.ID
}

// RemoveTask 删除定时任务
func (ht *HighPrecisionTimer) RemoveTask(taskID TimerTaskID) bool {
	ht.mu.Lock()
	defer ht.mu.Unlock()

	task, ok := ht.tasks[taskID]
	if !ok || !ht.running {
		LogWarnf("删除任务失败：任务不存在/引擎未运行，任务ID：%s", taskID)
		return false
	}

	// 取消任务上下文
	if task.cancel != nil {
		task.cancel()
	}

	// 从堆中删除（重新排序）
	for i, t := range ht.heap {
		if t.ID == taskID {
			heap.Remove(&ht.heap, i)
			break
		}
	}

	// 从索引中删除
	delete(ht.tasks, taskID)

	LogDebugf("任务删除成功：ID=%s", taskID)

	return true
}

// run 定时器主循环
func (ht *HighPrecisionTimer) run() {
	defer ht.wg.Done()

	for {
		ht.mu.Lock()

		// 退出条件
		if !ht.running || ht.ctx.Err() != nil {
			ht.mu.Unlock()
			return
		}

		// 无任务
		if ht.heap.Len() == 0 {
			ht.mu.Unlock()
			select {
			case <-time.After(1 * time.Millisecond):
			case <-ht.ctx.Done():
				return
			}
			continue
		}

		// 最近任务
		nextTask := ht.heap[0]
		now := time.Now()
		delay := nextTask.At.Sub(now)

		// 触发
		if delay <= 0 {
			heap.Pop(&ht.heap)
			delete(ht.tasks, nextTask.ID)
			ht.mu.Unlock()

			// 跳过已取消
			if nextTask.Ctx.Err() != nil {
				continue
			}

			// ====================== 通知观察者（解耦关键） ======================
			ht.observerMu.RLock()
			obs := ht.observer
			ht.observerMu.RUnlock()
			if obs != nil {
				obs.OnTimerTrigger(nextTask)
			}

			// 循环任务重新入堆
			ht.handleRepeatTask(nextTask, now)
			continue
		}

		// 设置定时器等待
		if ht.timer != nil {
			ht.timer.Stop()
		}
		ht.timer = time.NewTimer(delay)
		timer := ht.timer
		ht.mu.Unlock()

		select {
		case <-timer.C:
		case <-ht.wakeup:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		case <-ht.ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return
		}
	}
}

// handleRepeatTask 处理循环任务（Interval / Cron）
func (ht *HighPrecisionTimer) handleRepeatTask(task *TimerTask, now time.Time) {
	if task.Ctx.Err() != nil {
		return
	}

	var nextAt time.Time
	loop := false

	if task.Interval > 0 {
		nextAt = now.Add(task.Interval)
		loop = true
	} else if task.Cron != "" {
		if expr, err := cronexpr.Parse(task.Cron); err == nil {
			nextAt = expr.Next(now)
			loop = true
		}
	}

	if !loop || nextAt.IsZero() {
		return
	}

	newTask := &TimerTask{
		ID:       task.ID,
		At:       nextAt,
		Interval: task.Interval,
		Cron:     task.Cron,
		Priority: task.Priority,
		Callback: task.Callback,
		Ctx:      task.Ctx,
		cancel:   task.cancel,
	}

	_ = ht.AddTask(newTask)
}

// OnTimerTrigger 默认实现
func (ht *HighPrecisionTimer) OnTimerTrigger(task *TimerTask) {
	if task != nil && task.Callback != nil {
		err := task.Callback(task.Ctx)
		if err != nil {
			LogWarnf("定时任务回调执行失败：%v，任务ID：%s", err, task.ID)
		}
	}
}
