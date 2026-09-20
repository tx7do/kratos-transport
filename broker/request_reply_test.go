package broker

import (
	"context"
	"sync"
	"testing"
	"time"
)

// inMemBroker 进程内内存 Broker，用于验证 GenericRequest 的闭环逻辑
type inMemBroker struct {
	mu        sync.Mutex
	subs      map[string][]*inMemSubscriber
	published []string
}

type inMemSubscriber struct {
	topic   string
	handler Handler
	closed  bool
}

func newInMemBroker() *inMemBroker {
	return &inMemBroker{subs: make(map[string][]*inMemSubscriber)}
}

func (b *inMemBroker) Name() string         { return "inmem" }
func (b *inMemBroker) Options() Options     { return NewOptions() }
func (b *inMemBroker) Address() string      { return "memory" }
func (b *inMemBroker) Init(...Option) error { return nil }
func (b *inMemBroker) Connect() error       { return nil }
func (b *inMemBroker) Disconnect() error    { return nil }

func (b *inMemBroker) Request(ctx context.Context, topic string, msg *Message, opts ...RequestOption) (*Message, error) {
	return GenericRequest(ctx, b, topic, msg, opts...)
}

func (b *inMemBroker) Publish(ctx context.Context, topic string, msg *Message, _ ...PublishOption) error {
	b.mu.Lock()
	subs := append([]*inMemSubscriber(nil), b.subs[topic]...)
	b.mu.Unlock()

	for _, s := range subs {
		if s.closed {
			continue
		}
		_ = s.handler(ctx, &publication{topic: topic, message: msg})
	}
	return nil
}

func (b *inMemBroker) Subscribe(topic string, handler Handler, _ Binder, _ ...SubscribeOption) (Subscriber, error) {
	s := &inMemSubscriber{topic: topic, handler: handler}
	b.mu.Lock()
	b.subs[topic] = append(b.subs[topic], s)
	b.mu.Unlock()
	return s, nil
}

func (s *inMemSubscriber) Options() SubscribeOptions { return NewSubscribeOptions() }
func (s *inMemSubscriber) Topic() string             { return s.topic }
func (s *inMemSubscriber) Unsubscribe(bool) error {
	s.closed = true
	return nil
}

// publication 最小 Event 实现
type publication struct {
	topic   string
	message *Message
}

func (p *publication) Topic() string     { return p.topic }
func (p *publication) Message() *Message { return p.message }
func (p *publication) RawMessage() any   { return nil }
func (p *publication) Ack() error        { return nil }
func (p *publication) Error() error      { return nil }

func TestGenericRequestRoundTrip(t *testing.T) {
	b := newInMemBroker()

	// 应答方：收到请求后 ReplyTo
	handler := func(ctx context.Context, event Event) error {
		req := event.Message()
		if req == nil {
			return nil
		}
		return ReplyTo(ctx, b, req, map[string]string{"echo": "pong"})
	}
	if _, err := b.Subscribe("req.topic", handler, nil); err != nil {
		t.Fatal(err)
	}

	go func() {
		_, _ = GenericRequest(context.Background(), b, "req.topic", NewMessage("ping"), WithRequestTimeout(3*time.Second))
	}()

	// 请求方
	reply, err := GenericRequest(context.Background(), b, "req.topic", NewMessage("ping"),
		WithRequestTimeout(3*time.Second))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, ok := reply.Body.(map[string]string)
	if !ok || body["echo"] != "pong" {
		t.Fatalf("unexpected reply body: %#v", reply.Body)
	}
	if reply.GetHeader(HeaderCorrelationID) == "" {
		t.Fatal("reply missing correlation id")
	}
}

func TestGenericRequestCorrelation(t *testing.T) {
	b := newInMemBroker()

	// 应答方：只回应带 Reply-To 的消息，并原样回传 Correlation-ID
	handler := func(ctx context.Context, event Event) error {
		req := event.Message()
		if req == nil {
			return nil
		}
		return ReplyTo(ctx, b, req, "ack")
	}
	if _, err := b.Subscribe("req.topic", handler, nil); err != nil {
		t.Fatal(err)
	}

	reply, err := GenericRequest(context.Background(), b, "req.topic", NewMessage("m1"),
		WithRequestTimeout(3*time.Second))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if reply.Body != "ack" {
		t.Fatalf("unexpected body: %v", reply.Body)
	}
}

func TestGenericRequestTimeout(t *testing.T) {
	b := newInMemBroker() // 无应答方

	start := time.Now()
	_, err := GenericRequest(context.Background(), b, "req.topic", NewMessage("ping"),
		WithRequestTimeout(300*time.Millisecond))
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("timeout took too long: %v", elapsed)
	}
}
