package thrift

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"testing"

	api "github.com/tx7do/kratos-transport/testing/api/thrift/gen-go/hygrothermograph"
)

type HygrothermographHandler struct {
}

func NewHygrothermographHandler() *HygrothermographHandler {
	return &HygrothermographHandler{}
}

func (p *HygrothermographHandler) GetHygrothermograph(_ context.Context) (_r *api.Hygrothermograph, _err error) {
	var Humidity = float64(rand.Intn(100))
	var Temperature = float64(rand.Intn(100))
	_r = &api.Hygrothermograph{
		Humidity:    &Humidity,
		Temperature: &Temperature,
	}
	fmt.Println("Humidity:", Humidity, "Temperature:", Temperature)
	return
}

func TestServer(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		WithAddress(":7700"),
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
