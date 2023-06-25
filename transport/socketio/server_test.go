package socketio

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	socketio "github.com/googollee/go-socket.io"
)

func TestServer(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		WithAddress(":8000"),
		WithCodec("json"),
		WithPath("/socket.io/"),
	)

	srv.RegisterConnectHandler("/", func(s socketio.Conn) error {
		s.SetContext("")
		log.Info("connected:", s.ID())
		return nil
	})

	srv.RegisterEventHandler("/", "notice", func(s socketio.Conn, msg string) {
		log.Info("notice:", msg)
		s.Emit("reply", "have "+msg)
	})

	srv.RegisterEventHandler("/chat", "msg", func(s socketio.Conn, msg string) string {
		s.SetContext(msg)
		return "recv " + msg
	})

	srv.RegisterEventHandler("/", "bye", func(s socketio.Conn) string {
		last := s.Context().(string)
		s.Emit("bye", last)
		_ = s.Close()
		return last
	})

	srv.RegisterErrorHandler("/", func(s socketio.Conn, e error) {
		log.Info("meet error:", e)
	})

	srv.RegisterDisconnectHandler("/", func(s socketio.Conn, reason string) {
		log.Info("closed", reason)
	})

	if err := srv.Start(ctx); err != nil {
		panic(err)
	}

	defer func() {
		if err := srv.Stop(ctx); err != nil {
			t.Errorf("expected nil got %v", err)
		}
	}()

	<-interrupt
}
