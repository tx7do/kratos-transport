package thrift

import (
	"context"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"testing"

	api "github.com/tx7do/kratos-transport/_example/api/thrift/gen-go"
)

type HygrothermographHandler struct {
}

func NewHygrothermographHandler() *HygrothermographHandler {
	return &HygrothermographHandler{}
}

func (p *HygrothermographHandler) GetHygrothermograph(ctx context.Context) (_r *api.Hygrothermograph, _err error) {
	_r.Humidity = float64(rand.Intn(100))
	_r.Temperature = float64(rand.Intn(100))
	return _r, nil
}

func TestServer(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		WithAddress(":8800"),
		WithProcessor(api.NewHygrothermographServiceProcessor(NewHygrothermographHandler())),
	)

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
