package hptimer

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockTimerObserver struct {
	triggeredTasks map[TimerTaskID]int
	mu             sync.Mutex
}

func (m *mockTimerObserver) OnTimerTrigger(task *TimerTask) {
	m.mu.Lock()
	m.triggeredTasks[task.ID]++
	m.mu.Unlock()
	if task.Callback != nil {
		_ = task.Callback(task.Ctx)
	}
}

func newMockTimerObserver() *mockTimerObserver {
	return &mockTimerObserver{
		triggeredTasks: make(map[TimerTaskID]int),
	}
}

// absDur 返回 duration 的绝对值
func absDur(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

// TestHighPrecisionTimer_AddSingleTask 测试单次定时任务触发
func TestHighPrecisionTimer_AddSingleTask(t *testing.T) {
	assertions := assert.New(t)

	observer := newMockTimerObserver()
	ht := NewHighPrecisionTimer(observer)
	assert.NotNil(t, ht)
	ht.Start()
	defer ht.Stop()

	// 使用 channel 等待触发
	triggerCh := make(chan time.Time, 1)

	// 定义单次任务（50ms后触发，放宽到 50ms）
	taskID := TimerTaskID("single_task")
	triggerTime := time.Now().Add(50 * time.Millisecond)
	var actualTriggerTime time.Time

	// 添加单次任务
	ht.AddTask(&TimerTask{
		ID:       taskID,
		At:       triggerTime,
		Interval: 0, // 单次任务
		Priority: 0, // 可自定义优先级
		Callback: func(ctx context.Context) error {
			// 将触发时间发送到 channel（非阻塞）
			select {
			case triggerCh <- time.Now():
			default:
			}
			return nil
		},
	})

	// 等待任务触发（超时控制：600ms）
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	select {
	case actualTriggerTime = <-triggerCh:
		// 验证任务触发时间误差（允许±40ms）
		delta := actualTriggerTime.Sub(triggerTime)
		assertions.LessOrEqual(absDur(delta), 40*time.Millisecond, "单次任务触发误差过大")
	case <-ctx.Done():
		t.Fatal("单次任务未按时触发，超时")
	}

	// 验证任务触发后被移除（堆为空）
	ht.mu.Lock()
	assertions.Equal(0, ht.heap.Len(), "单次任务触发后应从堆中移除")
	assertions.NotContains(ht.tasks, taskID, "单次任务触发后应从索引中移除")
	ht.mu.Unlock()
}

// TestHighPrecisionTimer_AddLoopTask 测试循环定时任务触发
func TestHighPrecisionTimer_AddLoopTask(t *testing.T) {
	assertions := assert.New(t)

	// 1. 初始化模拟EventLoop
	observer := newMockTimerObserver()
	ht := NewHighPrecisionTimer(observer)
	assert.NotNil(t, ht)
	ht.Start()
	defer ht.Stop()

	// 3. 测试配置
	const expectCount = 2
	var (
		wg        sync.WaitGroup
		triggered bool
		mu        sync.Mutex
	)
	wg.Add(expectCount)

	// 4. 定义循环任务
	taskID := TimerTaskID("test_loop_task")
	task := &TimerTask{
		ID:       taskID,
		At:       time.Now().Add(100 * time.Millisecond), // 首次触发延迟100ms
		Interval: 100 * time.Millisecond,                 // 循环间隔100ms
		Priority: 0,                                      // 可自定义优先级
		Callback: func(ctx context.Context) error {
			mu.Lock()
			triggered = true
			mu.Unlock()
			wg.Done()
			return nil
		},
	}

	// 5. 添加任务（检查返回值）
	addedTaskID := ht.AddTask(task)
	assertions.Equal(taskID, addedTaskID, "添加任务失败")

	// 6. 等待任务触发（超时放宽到1000ms）
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()

	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()

	// 7. 验证结果
	select {
	case <-doneCh:
		// 检查EventLoop中触发的次数
		observer.mu.Lock()
		triggerCount := observer.triggeredTasks[taskID]
		observer.mu.Unlock()

		assertions.Equal(expectCount, triggerCount,
			fmt.Sprintf("期望触发%d次，实际%d次", expectCount, triggerCount))
		assertions.True(triggered, "任务回调未执行")

		// 清理任务
		removed := ht.RemoveTask(taskID)
		assertions.True(removed, "删除任务失败")

	case <-ctx.Done():
		// 打印调试日志
		observer.mu.Lock()
		triggerCount := observer.triggeredTasks[taskID]
		observer.mu.Unlock()

		t.Fatalf(
			"循环任务触发超时（%v）！已触发%d次（EventLoop），回调触发：%t，期望%d次",
			ctx.Err(), triggerCount, triggered, expectCount,
		)
	}
}

// TestHighPrecisionTimer_RemoveTask 测试删除未触发的任务
func TestHighPrecisionTimer_RemoveTask(t *testing.T) {
	assertions := assert.New(t)
	observer := newMockTimerObserver()
	ht := NewHighPrecisionTimer(observer)
	assert.NotNil(t, ht)
	ht.Start()
	defer ht.Stop()

	// 标记任务是否触发
	triggered := false
	var mu sync.Mutex

	// 添加延迟任务（50ms后触发）
	taskID := TimerTaskID("remove_task")
	ht.AddTask(&TimerTask{
		ID:       taskID,
		At:       time.Now().Add(50 * time.Millisecond),
		Interval: 0,
		Priority: 0, // 可自定义优先级
		Callback: func(ctx context.Context) error {
			mu.Lock()
			triggered = true
			mu.Unlock()
			return nil
		},
	})

	// 立即删除任务
	success := ht.RemoveTask(taskID)
	assertions.True(success, "删除任务应返回成功")

	// 等待足够时间，验证任务未触发
	time.Sleep(60 * time.Millisecond)

	mu.Lock()
	assertions.False(triggered, "已删除的任务不应触发")
	mu.Unlock()

	// 验证任务从堆和索引中移除
	ht.mu.Lock()
	assertions.NotContains(ht.tasks, taskID)
	assertions.Equal(0, ht.heap.Len())
	ht.mu.Unlock()
}

// TestHighPrecisionTimer_Stop 测试停止引擎后任务不再触发
func TestHighPrecisionTimer_Stop(t *testing.T) {
	assertions := assert.New(t)
	observer := newMockTimerObserver()
	ht := NewHighPrecisionTimer(observer)
	assert.NotNil(t, ht)
	ht.Start()
	defer ht.Stop()

	// 标记任务是否触发
	triggered := false

	// 添加任务（10ms后触发）
	ht.AddTask(&TimerTask{
		ID:       "stop_task",
		At:       time.Now().Add(10 * time.Millisecond),
		Interval: 0,
		Priority: 0, // 可自定义优先级
		Callback: func(ctx context.Context) error {
			triggered = true
			return nil
		},
	})

	// 立即停止定时器引擎
	ht.Stop()

	// 等待足够时间，验证任务未触发
	time.Sleep(20 * time.Millisecond)
	assertions.False(triggered, "停止引擎后任务不应触发")

	// 验证引擎状态
	assertions.False(ht.running, "引擎停止后running应为false")
}

// TestHighPrecisionTimer_CancelByCtx 测试通过上下文取消任务
func TestHighPrecisionTimer_CancelByCtx(t *testing.T) {
	assertions := assert.New(t)
	observer := newMockTimerObserver()
	ht := NewHighPrecisionTimer(observer)
	assert.NotNil(t, ht)
	ht.Start()
	defer ht.Stop()

	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 标记任务是否触发
	triggered := false

	// 添加带上下文的任务
	taskID := TimerTaskID("ctx_cancel_task")
	ht.AddTask(&TimerTask{
		ID:       taskID,
		At:       time.Now().Add(20 * time.Millisecond),
		Interval: 0,
		Priority: 0,   // 可自定义优先级
		Ctx:      ctx, // 绑定上下文
		Callback: func(ctx context.Context) error {
			triggered = true
			return nil
		},
	})

	// 取消上下文
	cancel()

	// 等待足够时间，验证任务未触发
	time.Sleep(30 * time.Millisecond)
	assertions.False(triggered, "上下文取消的任务不应触发")
}

// TestHighPrecisionTimer_ConcurrentAddRemove 测试并发添加/删除任务（线程安全）
func TestHighPrecisionTimer_ConcurrentAddRemove(t *testing.T) {
	assertions := assert.New(t)
	observer := newMockTimerObserver()
	ht := NewHighPrecisionTimer(observer)
	assert.NotNil(t, ht)
	ht.Start()
	defer ht.Stop()

	// 并发添加/删除100个任务
	var wg sync.WaitGroup
	wg.Add(100)
	for i := 0; i < 100; i++ {
		go func(idx int) {
			defer wg.Done()
			taskID := TimerTaskID(fmt.Sprintf("concurrent_%d", idx))
			// 添加任务
			ht.AddTask(&TimerTask{
				ID:       taskID,
				At:       time.Now().Add(100 * time.Millisecond),
				Interval: 0,
				Priority: 0, // 可自定义优先级
			})
			// 立即删除
			ht.RemoveTask(taskID)
		}(i)
	}

	// 等待并发操作完成
	wg.Wait()

	// 验证堆和索引为空
	ht.mu.Lock()
	assertions.Equal(0, ht.heap.Len())
	assertions.Equal(0, len(ht.tasks))
	ht.mu.Unlock()
}

// TestHighPrecisionTimer_AddRemove_Stop 验证 AddTask 返回 ID、RemoveTask 能删除任务，并能正常 Stop
func TestHighPrecisionTimer_AddRemove_Stop(t *testing.T) {
	observer := newMockTimerObserver()
	ht := NewHighPrecisionTimer(observer)
	assert.NotNil(t, ht)
	ht.Start()
	defer ht.Stop()

	// 创建一个远期任务，确保不会在测试期间触发
	task := NewTimerTask("t1", time.Now().Add(1*time.Hour), WithData("payload"))
	id := ht.AddTask(task)
	if id == "" {
		ht.Stop()
		t.Fatalf("AddTask 返回空 ID")
	}
	if id != "t1" {
		ht.Stop()
		t.Fatalf("AddTask 返回 ID 不匹配，期望 %s，实际 %s", "t1", id)
	}

	// 删除任务
	if !ht.RemoveTask(id) {
		ht.Stop()
		t.Fatalf("RemoveTask 未能删除任务 %s", id)
	}

	ht.Stop()
}

// TestHighPrecisionTimer_Cron_SetsAtAndStopCancels 验证当使用 Cron 且 At 为空时，AddTask 会设置首次触发时间；Stop 会取消任务上下文
func TestHighPrecisionTimer_Cron_SetsAtAndStopCancels(t *testing.T) {
	observer := newMockTimerObserver()
	ht := NewHighPrecisionTimer(observer)
	assert.NotNil(t, ht)
	ht.Start()
	defer ht.Stop()

	// 使用每秒触发的 cron 表达式（依赖 github.com/gorhill/cronexpr 的解析格式）
	cronExpr := "*/1 * * * * *"
	task := NewTimerTask("cron1", time.Time{}, WithCron(cronExpr))
	id := ht.AddTask(task)
	if id == "" {
		ht.Stop()
		t.Fatalf("AddTask 返回空 ID，可能 cron 解析失败")
	}

	// AddTask 应该为 task 设置 At（首次触发时间）
	if task.At.IsZero() {
		ht.Stop()
		t.Fatalf("期望 AddTask 为 Cron 任务设置首次触发时间 At，但 At 仍为零值")
	}

	// Stop 应该取消任务上下文
	ht.Stop()
	// 允许 goroutine 处理取消
	time.Sleep(10 * time.Millisecond)
	if task.Ctx == nil {
		t.Fatalf("任务 Context 为 nil，期望非 nil")
	}
	if task.Ctx.Err() == nil {
		t.Fatalf("Stop 后期望任务上下文被取消，Ctx.Err() 应为非 nil")
	}
}

// Test: 在定时回调中重新下一个更早的任务，应被及时触发
func TestHighPrecisionTimer_CallbackReaddEarlierTask(t *testing.T) {
	assertions := assert.New(t)

	observer := newMockTimerObserver()
	ht := NewHighPrecisionTimer(observer)
	assert.NotNil(t, ht)
	ht.Start()
	defer ht.Stop()

	bTriggered := make(chan struct{}, 1)

	// 定义 A: 在触发时添加 B（B 比 A 更早）
	taskA := &TimerTask{
		ID:       TimerTaskID("taskA_readd"),
		At:       time.Now().Add(50 * time.Millisecond),
		Interval: 0,
		Priority: 0, // 可自定义优先级
		Callback: func(ctx context.Context) error {
			// 在回调中添加 B，B 的触发时间是现在+10ms（应尽快被调度）
			taskB := &TimerTask{
				ID:       TimerTaskID("taskB_readd"),
				At:       time.Now().Add(10 * time.Millisecond),
				Interval: 0,
				Priority: 0, // 可自定义优先级
				Callback: func(ctx context.Context) error {
					select {
					case bTriggered <- struct{}{}:
					default:
					}
					return nil
				},
			}
			id := ht.AddTask(taskB)
			if id == "" {
				t.Logf("AddTask 返回空 ID（回调内添加），可能失败")
			}
			return nil
		},
	}

	_ = ht.AddTask(taskA)

	// 等待 B 被触发
	select {
	case <-bTriggered:
		// success
	case <-time.After(500 * time.Millisecond):
		t.Fatal("回调中重新添加的更早任务未在预期时间内触发")
	}

	// 清理（确保堆无遗留）
	ht.mu.Lock()
	assertions.GreaterOrEqual(0, ht.heap.Len())
	ht.mu.Unlock()
}

// Test: 在定时回调中删除另一个挂起任务，删除的任务不应触发
func TestHighPrecisionTimer_CallbackRemovePendingTask(t *testing.T) {
	assertions := assert.New(t)

	observer := newMockTimerObserver()
	ht := NewHighPrecisionTimer(observer)
	assert.NotNil(t, ht)
	ht.Start()
	defer ht.Stop()

	xTriggered := make(chan struct{}, 1)
	yTriggered := make(chan struct{}, 1)

	// X: 计划较晚触发（应被 Y 删除）
	taskX := &TimerTask{
		ID:       TimerTaskID("taskX_remove"),
		At:       time.Now().Add(150 * time.Millisecond),
		Interval: 0,
		Priority: 0, // 可自定义优先级
		Callback: func(ctx context.Context) error {
			select {
			case xTriggered <- struct{}{}:
			default:
			}
			return nil
		},
	}
	_ = ht.AddTask(taskX)

	// Y: 较早触发，在回调中删除 X
	taskY := &TimerTask{
		ID:       TimerTaskID("taskY_remove"),
		At:       time.Now().Add(50 * time.Millisecond),
		Interval: 0,
		Priority: 0, // 可自定义优先级
		Callback: func(ctx context.Context) error {
			_ = ht.RemoveTask(taskX.ID)
			select {
			case yTriggered <- struct{}{}:
			default:
			}
			return nil
		},
	}
	_ = ht.AddTask(taskY)

	// 等待 Y 触发，确保删除操作已执行
	select {
	case <-yTriggered:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("用于删除的任务 Y 未在预期时间触发")
	}

	// 再等待一段时间，验证 X 未触发
	select {
	case <-xTriggered:
		t.Fatal("被回调中删除的任务 X 不应触发")
	case <-time.After(200 * time.Millisecond):
		// success: X 未触发
	}

	// 验证任务从索引/堆中移除
	ht.mu.Lock()
	_, exists := ht.tasks[taskX.ID]
	assertions.False(exists, "任务 X 应从索引中移除")
	ht.mu.Unlock()
}

// Benchmark: 单任务高精度定时器触发
func BenchmarkHighPrecisionTimer_SingleTask(b *testing.B) {
	observer := newMockTimerObserver()
	ht := NewHighPrecisionTimer(observer)
	ht.Start()
	defer ht.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch := make(chan struct{}, 1)
		task := &TimerTask{
			ID: TimerTaskID(fmt.Sprintf("bench_single_%d", i)),
			At: time.Now().Add(1 * time.Millisecond),
			Callback: func(ctx context.Context) error {
				ch <- struct{}{}
				return nil
			},
		}
		ht.AddTask(task)
		<-ch
	}
}

// Benchmark: 批量任务高精度定时器触发
func BenchmarkHighPrecisionTimer_BatchTasks(b *testing.B) {
	observer := newMockTimerObserver()
	ht := NewHighPrecisionTimer(observer)
	ht.Start()
	defer ht.Stop()

	taskCount := 1000
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch := make(chan struct{}, taskCount)
		for j := 0; j < taskCount; j++ {
			task := &TimerTask{
				ID: TimerTaskID(fmt.Sprintf("bench_batch_%d_%d", i, j)),
				At: time.Now().Add(1 * time.Millisecond),
				Callback: func(ctx context.Context) error {
					ch <- struct{}{}
					return nil
				},
			}
			ht.AddTask(task)
		}
		for j := 0; j < taskCount; j++ {
			<-ch
		}
	}
}

// TestHighPrecisionTimer_ConcurrentSafety 测试高并发下的线程安全
func TestHighPrecisionTimer_ConcurrentSafety(t *testing.T) {
	observer := newMockTimerObserver()
	// 这里不提前 Stop，避免 Stop 影响 Add/Remove 的并发安全测试
	ht := NewHighPrecisionTimer(observer)
	ht.Start()

	taskCount := 1000
	var wg sync.WaitGroup
	wg.Add(taskCount * 2)

	// 并发添加任务
	for i := 0; i < taskCount; i++ {
		go func(idx int) {
			defer wg.Done()
			taskID := TimerTaskID(fmt.Sprintf("concurrent_safe_%d", idx))
			ht.AddTask(&TimerTask{
				ID:       taskID,
				At:       time.Now().Add(10 * time.Millisecond),
				Callback: func(ctx context.Context) error { return nil },
			})
		}(i)
	}

	// 并发删除任务
	for i := 0; i < taskCount; i++ {
		go func(idx int) {
			defer wg.Done()
			taskID := TimerTaskID(fmt.Sprintf("concurrent_safe_%d", idx))
			ht.RemoveTask(taskID)
		}(i)
	}

	wg.Wait()

	// Stop 放在所有 Add/Remove 完成后，保证所有操作都已完成
	ht.Stop()

	// 验证不会 panic，且所有任务都被安全处理
	ht.mu.Lock()
	if len(ht.tasks) != 0 {
		t.Errorf("期望所有任务都被安全移除，实际剩余: %d", len(ht.tasks))
	}
	if ht.heap.Len() != 0 {
		t.Errorf("期望堆为空，实际: %d", ht.heap.Len())
	}
	ht.mu.Unlock()
}

// Test: 在Callback中移除自身任务，验证不会panic且行为安全
func TestHighPrecisionTimer_CallbackRemoveSelf(t *testing.T) {
	observer := newMockTimerObserver()
	ht := NewHighPrecisionTimer(observer)
	ht.Start()
	defer ht.Stop()

	triggered := make(chan struct{}, 1)
	taskID := TimerTaskID("callback_remove_self")
	task := &TimerTask{
		ID: taskID,
		At: time.Now().Add(10 * time.Millisecond),
		Callback: func(ctx context.Context) error {
			// 在回调中移除自身
			removed := ht.RemoveTask(taskID)
			// 允许移除失败（因为任务已被调度出堆），但不应panic
			select {
			case triggered <- struct{}{}:
			default:
			}
			if !removed {
				t.Logf("回调中移除自身任务未成功（可能已被调度出堆），允许此情况")
			}
			return nil
		},
	}
	ht.AddTask(task)

	select {
	case <-triggered:
		// success
	case <-time.After(200 * time.Millisecond):
		t.Fatal("回调中移除自身任务未被安全执行")
	}

	ht.mu.Lock()
	_, exists := ht.tasks[taskID]
	if exists {
		t.Errorf("任务应已被移除，实际仍存在")
	}
	if ht.heap.Len() != 0 {
		t.Errorf("堆应为空，实际: %d", ht.heap.Len())
	}
	ht.mu.Unlock()
}
