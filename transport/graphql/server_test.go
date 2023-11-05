package graphql

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"testing"

	api "github.com/tx7do/kratos-transport/testing/api/graphql"
)

type resolver struct{}

func (r *resolver) Query() api.QueryResolver {
	return &queryResolver{}
}

type queryResolver struct{}

func (r *queryResolver) Hygrothermograph(ctx context.Context) (*api.Hygrothermograph, error) {
	ret := &api.Hygrothermograph{
		Humidity:    float64(rand.Intn(100)),
		Temperature: float64(rand.Intn(100)),
	}
	fmt.Println("Humidity:", ret.Humidity, "Temperature:", ret.Temperature)
	return ret, nil
}

func TestServer(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		WithAddress(":8800"),
	)

	srv.Handle("/query", api.NewExecutableSchema(api.Config{Resolvers: &resolver{}}))

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
