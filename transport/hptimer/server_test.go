package hptimer

import (
	"context"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	// 创建 cron 服务
	srv := NewServer(
		WithEnableKeepAlive(false),
	)

	if err := srv.Start(t.Context()); err != nil {
		t.Fatalf("server start failed: %v", err)
	}

	// 注册一个高精度定时任务，触发后通过 channel 通知
	triggerCh := make(chan struct{}, 1)
	taskID := "test_task"
	innerTaskID := srv.AddTask(&TimerTask{
		ID:       TimerTaskID(taskID),
		At:       time.Now().Add(50 * time.Millisecond),
		Interval: 0,
		Callback: func(ctx context.Context) error {
			triggerCh <- struct{}{}
			return nil
		},
	})
	if innerTaskID == "" {
		t.Fatalf("AddTask failed: %v", taskID)
	}

	// 验证任务是否按时触发
	select {
	case <-triggerCh:
		// success
	case <-time.After(500 * time.Millisecond):
		t.Fatal("定时任务未按时触发")
	}

	// 删除任务（已触发应返回false）
	removed := srv.RemoveTask(TimerTaskID(taskID))
	if removed {
		t.Errorf("RemoveTask 应返回 false (已触发)")
	}

	// 优雅关闭
	if err := srv.Stop(t.Context()); err != nil {
		t.Errorf("expected nil got %v", err)
	}
}
