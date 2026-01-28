package broker

import (
	"context"
	"reflect"
	"testing"
)

func TestChainPublishMiddleware_Order(t *testing.T) {
	var seq []int

	base := PublishHandler(func(ctx context.Context, topic string, msg any, opts ...PublishOption) error {
		seq = append(seq, 2)
		return nil
	})

	mw1 := func(next PublishHandler) PublishHandler {
		return func(ctx context.Context, topic string, msg any, opts ...PublishOption) error {
			seq = append(seq, 1)
			return next(ctx, topic, msg, opts...)
		}
	}

	mw2 := func(next PublishHandler) PublishHandler {
		return func(ctx context.Context, topic string, msg any, opts ...PublishOption) error {
			seq = append(seq, 3)
			return next(ctx, topic, msg, opts...)
		}
	}

	chained := ChainPublishMiddleware(base, []PublishMiddleware{mw1, mw2})

	err := chained(context.Background(), "topic", &Message{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []int{1, 3, 2} // mw1 -> mw2 -> base
	if !reflect.DeepEqual(seq, want) {
		t.Fatalf("unexpected order: got %v want %v", seq, want)
	}
}
