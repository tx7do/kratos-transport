package broker

import (
	"context"
	"reflect"
	"testing"
)

// dummyEvent 用于满足 Event 接口，handler 中不会使用其字段
type dummyEvent struct{}

func (d dummyEvent) Topic() string     { return "topic" }
func (d dummyEvent) Message() *Message { return &Message{} }
func (d dummyEvent) RawMessage() any   { return nil }
func (d dummyEvent) Ack() error        { return nil }
func (d dummyEvent) Error() error      { return nil }

func TestChainSubscriberMiddleware_Order(t *testing.T) {
	var seq []int

	base := Handler(func(ctx context.Context, evt Event) error {
		seq = append(seq, 2)
		return nil
	})

	mw1 := func(next Handler) Handler {
		return func(ctx context.Context, evt Event) error {
			seq = append(seq, 1)
			return next(ctx, evt)
		}
	}

	mw2 := func(next Handler) Handler {
		return func(ctx context.Context, evt Event) error {
			seq = append(seq, 3)
			return next(ctx, evt)
		}
	}

	chained := ChainSubscriberMiddleware(base, []SubscriberMiddleware{mw1, mw2})

	if err := chained(context.Background(), dummyEvent{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []int{1, 3, 2} // mw1 -> mw2 -> base
	if !reflect.DeepEqual(seq, want) {
		t.Fatalf("unexpected order: got %v want %v", seq, want)
	}
}

func TestChainSubscriberMiddleware_SkipNil(t *testing.T) {
	var seq []int

	base := Handler(func(ctx context.Context, evt Event) error {
		seq = append(seq, 2)
		return nil
	})

	mw := func(next Handler) Handler {
		return func(ctx context.Context, evt Event) error {
			seq = append(seq, 1)
			return next(ctx, evt)
		}
	}

	// 第一个为 nil，应被跳过；实际顺序为 mw -> base
	chained := ChainSubscriberMiddleware(base, []SubscriberMiddleware{nil, mw})

	if err := chained(context.Background(), dummyEvent{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []int{1, 2}
	if !reflect.DeepEqual(seq, want) {
		t.Fatalf("unexpected order with nil middleware: got %v want %v", seq, want)
	}
}
