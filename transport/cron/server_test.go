package cron

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"testing"
)

func TestServer(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// 创建 cron 服务
	srv := NewServer(
		WithEnableKeepAlive(false),
	)

	if err := srv.Start(t.Context()); err != nil {
		panic(err)
	}

	// 每 10 秒执行一次
	_, _ = srv.StartTimerJob("*/10 * * * * *", func() {
		log.Println("task run every 10 seconds")
	})

	// 每分钟执行一次
	_, _ = srv.StartTimerJob("0 */1 * * * *", func() {
		log.Println("task run every minute")
	})

	defer func() {
		if err := srv.Stop(t.Context()); err != nil {
			t.Errorf("expected nil got %v", err)
		}
	}()

	<-interrupt
}
