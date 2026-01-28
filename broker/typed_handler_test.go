package broker

import (
	"context"
	"strings"
	"testing"
)

// 用于测试的具体类型
type MyType struct {
	V int
}

// 简单的 Event 测试实现
type testEvent struct {
	topic string
	msg   *Message
	raw   any
}

func (e testEvent) Topic() string     { return e.topic }
func (e testEvent) Message() *Message { return e.msg }
func (e testEvent) RawMessage() any   { return e.raw }
func (e testEvent) Ack() error        { return nil }
func (e testEvent) Error() error      { return nil }

func TestTypedToEventHandler_NilEvent(t *testing.T) {
	h := TypedToEventHandler(func(ctx context.Context, topic string, headers Headers, m *MyType) error {
		t.Fatalf("handler should not be called for nil event")
		return nil
	})

	if err := h(context.Background(), nil); err == nil || err.Error() != "evt is nil" {
		t.Fatalf("expected error 'evt is nil', got: %v", err)
	}
}

func TestTypedToEventHandler_NilMessageOrBody(t *testing.T) {
	handler := TypedToEventHandler(func(ctx context.Context, topic string, headers Headers, m *MyType) error {
		t.Fatalf("handler should not be called when message or body is nil")
		return nil
	})

	// 情况一：Message 为 nil
	ev1 := testEvent{topic: "t1", msg: nil}
	if err := handler(context.Background(), ev1); err == nil || err.Error() != "event message or body is nil" {
		t.Fatalf("expected 'event message or body is nil' for nil Message, got: %v", err)
	}

	// 情况二：Message 非 nil 但 Body 为 nil
	ev2 := testEvent{topic: "t2", msg: &Message{Headers: nil, Body: nil}}
	if err := handler(context.Background(), ev2); err == nil || err.Error() != "event message or body is nil" {
		t.Fatalf("expected 'event message or body is nil' for nil Body, got: %v", err)
	}
}

func TestTypedToEventHandler_HandlePointerAndValue(t *testing.T) {
	called := 0

	handler := TypedToEventHandler(func(ctx context.Context, topic string, headers Headers, m *MyType) error {
		called++
		// 验证传入的 topic/headers/值
		if topic != "topic1" && topic != "topic2" {
			t.Fatalf("unexpected topic: %s", topic)
		}
		if headers["h"] != "v" {
			t.Fatalf("unexpected headers: %v", headers)
		}
		if m == nil {
			t.Fatalf("expected non-nil *MyType")
		}
		if m.V != 1 && m.V != 2 {
			t.Fatalf("unexpected MyType.V: %d", m.V)
		}
		return nil
	})

	// 用指针作为 body
	evPtr := testEvent{
		topic: "topic1",
		msg:   &Message{Headers: Headers{"h": "v"}, Body: &MyType{V: 1}},
	}
	if err := handler(context.Background(), evPtr); err != nil {
		t.Fatalf("unexpected error for pointer body: %v", err)
	}

	// 用值作为 body
	evVal := testEvent{
		topic: "topic2",
		msg:   &Message{Headers: Headers{"h": "v"}, Body: MyType{V: 2}},
	}
	if err := handler(context.Background(), evVal); err != nil {
		t.Fatalf("unexpected error for value body: %v", err)
	}

	if called != 2 {
		t.Fatalf("handler should be called twice, called=%d", called)
	}
}

func TestTypedToEventHandler_UnsupportedType(t *testing.T) {
	handler := TypedToEventHandler(func(ctx context.Context, topic string, headers Headers, m *MyType) error {
		t.Fatalf("handler should not be called for unsupported body type")
		return nil
	})

	ev := testEvent{
		topic: "t",
		msg:   &Message{Headers: Headers{}, Body: "not-mytype"},
	}

	err := handler(context.Background(), ev)
	if err == nil {
		t.Fatalf("expected error for unsupported body type, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported body type") {
		t.Fatalf("unexpected error message: %v", err)
	}
}
