package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	kafkaGo "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/tracing"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	api "github.com/tx7do/kratos-transport/testing/api/manual"
)

const (
	testBrokers = "localhost:9092"

	testTopic         = "logger.sensor.ts"
	testWildCardTopic = "logger.sensor.+"

	testGroupId = "logger-group"
)

func handleHygrothermograph(_ context.Context, topic string, headers broker.Headers, msg *api.Hygrothermograph) error {
	LogInfof("Topic %s, Headers: %+v, Payload: %+v\n", topic, headers, msg)
	return nil
}

type HygrothermographHandler func(_ context.Context, topic string, headers broker.Headers, msg *api.Hygrothermograph) error

func RegisterHygrothermographHandler(fnc HygrothermographHandler) broker.Handler {
	return func(ctx context.Context, event broker.Event) error {
		var msg api.Hygrothermograph

		switch t := event.Message().Body.(type) {
		case []byte:
			if err := json.Unmarshal(t, &msg); err != nil {
				log.Error("json Unmarshal failed: ", err.Error())
				return err
			}
		case string:
			if err := json.Unmarshal([]byte(t), &msg); err != nil {
				log.Error("json Unmarshal failed: ", err.Error())
				return err
			}

		case *api.Hygrothermograph:
			msg = *t

		default:
			log.Error("unknown type Unmarshal failed: ", t)
			return fmt.Errorf("unsupported type: %T", t)
		}

		if err := fnc(ctx, event.Topic(), event.Message().Headers, &msg); err != nil {
			return err
		}

		return nil
	}
}

func Test_Publish_WithRawData(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	b := NewBroker(
		broker.WithAddress(testBrokers),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		t.Logf("cant connect to broker, skip: %v", err)
		t.Skip()
	}
	defer b.Disconnect()

	var msg api.Hygrothermograph
	const count = 10
	for i := 0; i < count; i++ {
		startTime := time.Now()
		msg.Humidity = float64(rand.Intn(100))
		msg.Temperature = float64(rand.Intn(100))
		buf, _ := json.Marshal(&msg)
		err := b.Publish(ctx, testTopic, broker.NewMessage(buf))
		assert.Nil(t, err)
		elapsedTime := time.Since(startTime) / time.Millisecond
		fmt.Printf("Publish %d, elapsed time: %dms, Humidity: %.2f Temperature: %.2f\n",
			i, elapsedTime, msg.Humidity, msg.Temperature)
	}

	fmt.Printf("total send %d messages\n", count)

	<-interrupt
}

func Test_Subscribe_WithRawData(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	b := NewBroker(
		broker.WithAddress(testBrokers),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		t.Logf("cant connect to broker, skip: %v", err)
		t.Skip()
	}
	defer b.Disconnect()

	_, err := b.Subscribe(testTopic,
		RegisterHygrothermographHandler(handleHygrothermograph),
		nil,
		broker.WithSubscribeQueueName(testGroupId),
	)
	assert.Nil(t, err)
	assert.Nil(t, err)

	<-interrupt
}

func Test_Publish_WithJsonCodec(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	b := NewBroker(
		broker.WithAddress(testBrokers),
		broker.WithCodec("json"),
		//WithAsync(false),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		t.Logf("cant connect to broker, skip: %v", err)
		t.Skip()
	}
	defer b.Disconnect()

	var headers broker.Headers
	headers = make(broker.Headers)
	headers["version"] = "1.0.0"

	var msg api.Hygrothermograph
	const count = 10
	for i := 0; i < count; i++ {
		startTime := time.Now()
		headers["trace_id"] = fmt.Sprintf("%d", i)
		msg.Humidity = float64(rand.Intn(100))
		msg.Temperature = float64(rand.Intn(100))
		err := b.Publish(ctx, testTopic, broker.NewMessage(msg, broker.WithHeaders(headers)))
		assert.Nil(t, err)
		elapsedTime := time.Since(startTime) / time.Millisecond
		t.Logf("Publish %d, elapsed time: %dms, Humidity: %.2f Temperature: %.2f\n",
			i, elapsedTime, msg.Humidity, msg.Temperature)
	}

	t.Logf("total send %d messages\n", count)

	<-interrupt
}

func Test_Subscribe_WithJsonCodec(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	b := NewBroker(
		broker.WithAddress(testBrokers),
		broker.WithCodec("json"),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		t.Logf("cant connect to broker, skip: %v", err)
		t.Skip()
	}
	defer b.Disconnect()

	_, err := b.Subscribe(
		testTopic,
		RegisterHygrothermographHandler(handleHygrothermograph),
		api.HygrothermographCreator,
		broker.WithSubscribeQueueName(testGroupId),
	)
	assert.Nil(t, err)

	<-interrupt
}

func createTracerProvider(exporterName, serviceName string) broker.Option {
	switch exporterName {
	case "otlp-grpc":
		return broker.WithTracerProvider(
			tracing.NewTracerProvider(exporterName,
				"localhost:4317",
				serviceName,
				"",
				"1.0.0",
				1.0,
			),
		)
	case "zipkin":
		return broker.WithTracerProvider(
			tracing.NewTracerProvider(exporterName,
				"http://localhost:9411/api/v2/spans",
				serviceName,
				"test",
				"1.0.0",
				1.0,
			),
		)
	}

	return nil
}

func Test_Publish_WithTracer(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	b := NewBroker(
		broker.WithAddress(testBrokers),
		broker.WithCodec("json"),
		createTracerProvider("otlp-grpc", "tracer_tester"),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		t.Logf("cant connect to broker, skip: %v", err)
		t.Skip()
	}
	defer b.Disconnect()

	var msg api.Hygrothermograph
	const count = 1
	for i := 0; i < count; i++ {
		startTime := time.Now()
		msg.Humidity = float64(rand.Intn(100))
		msg.Temperature = float64(rand.Intn(100))
		err := b.Publish(ctx, testTopic, broker.NewMessage(msg))
		assert.Nil(t, err)
		elapsedTime := time.Since(startTime) / time.Millisecond
		t.Logf("Publish %d, elapsed time: %dms, Humidity: %.2f Temperature: %.2f\n",
			i, elapsedTime, msg.Humidity, msg.Temperature)
	}

	t.Logf("total send %d messages\n", count)

	<-interrupt
}

func Test_Subscribe_WithTracer(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	b := NewBroker(
		broker.WithAddress(testBrokers),
		broker.WithCodec("json"),
		createTracerProvider("otlp-grpc", "subscribe_tracer_tester"),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		t.Logf("cant connect to broker, skip: %v", err)
		t.Skip()
	}
	defer b.Disconnect()

	_, err := b.Subscribe(testTopic,
		RegisterHygrothermographHandler(handleHygrothermograph),
		api.HygrothermographCreator,
		broker.WithSubscribeQueueName(testGroupId),
	)
	assert.Nil(t, err)

	<-interrupt
}

func Test_Publish_WithGlobalTracer(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	tp := tracing.NewTracerProvider(
		"otlp-grpc",
		"localhost:4317",
		"global_tracer_tester",
		"",
		"1.0.0",
		1.0,
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	defer func() {
		_ = tp.Shutdown(ctx)
	}()

	b := NewBroker(
		broker.WithAddress(testBrokers),
		broker.WithCodec("json"),
		broker.WithGlobalTracerProvider(),
		broker.WithGlobalPropagator(),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		t.Logf("cant connect to broker, skip: %v", err)
		t.Skip()
	}
	defer b.Disconnect()

	var msg api.Hygrothermograph
	const count = 1
	for i := 0; i < count; i++ {
		startTime := time.Now()
		msg.Humidity = float64(rand.Intn(100))
		msg.Temperature = float64(rand.Intn(100))
		err := b.Publish(ctx, testTopic, broker.NewMessage(msg))
		assert.Nil(t, err)
		elapsedTime := time.Since(startTime) / time.Millisecond
		t.Logf("Publish %d, elapsed time: %dms, Humidity: %.2f Temperature: %.2f\n",
			i, elapsedTime, msg.Humidity, msg.Temperature)
	}

	t.Logf("total send %d messages\n", count)

	<-interrupt
}

func Test_Subscribe_WithGlobalTracer(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	b := NewBroker(
		broker.WithAddress(testBrokers),
		broker.WithCodec("json"),
		broker.WithGlobalTracerProvider(),
		broker.WithGlobalPropagator(),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		t.Logf("cant connect to broker, skip: %v", err)
		t.Skip()
	}
	defer b.Disconnect()

	_, err := b.Subscribe(testTopic,
		RegisterHygrothermographHandler(handleHygrothermograph),
		api.HygrothermographCreator,
		broker.WithSubscribeQueueName(testGroupId),
	)
	assert.Nil(t, err)

	<-interrupt
}

func Test_Publish_WithCompletion(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	b := NewBroker(
		broker.WithAddress(testBrokers),
		broker.WithCodec("json"),
		createTracerProvider("otlp-grpc", "subscribe_tracer_tester"),
		WithAsync(true),
		WithCompletion(func(messages []kafkaGo.Message, err error) {
			t.Logf("send message complete: %v", err)
		}),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		t.Logf("cant connect to broker, skip: %v", err)
		t.Skip()
	}
	defer b.Disconnect()

	var msg api.Hygrothermograph
	const count = 1
	for i := 0; i < count; i++ {
		startTime := time.Now()
		msg.Humidity = float64(rand.Intn(100))
		msg.Temperature = float64(rand.Intn(100))
		err := b.Publish(ctx, testTopic, broker.NewMessage(msg))
		assert.Nil(t, err)
		elapsedTime := time.Since(startTime) / time.Millisecond
		t.Logf("Publish %d, elapsed time: %dms, Humidity: %.2f Temperature: %.2f\n",
			i, elapsedTime, msg.Humidity, msg.Temperature)
	}

	t.Logf("total send %d messages\n", count)

	<-interrupt
}

func Test_Subscribe_WithWildcardTopic(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	b := NewBroker(
		broker.WithAddress(testBrokers),
		broker.WithCodec("json"),
		createTracerProvider("otlp-grpc", "subscribe_tracer_tester"),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		t.Logf("cant connect to broker, skip: %v", err)
		t.Skip()
	}
	defer b.Disconnect()

	_, err := b.Subscribe(testWildCardTopic,
		RegisterHygrothermographHandler(handleHygrothermograph),
		api.HygrothermographCreator,
		broker.WithSubscribeQueueName(testGroupId),
	)
	assert.Nil(t, err)

	<-interrupt
}

func Test_Subscribe_Batch(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	b := NewBroker(
		broker.WithAddress(testBrokers),
		broker.WithCodec("json"),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		t.Logf("cant connect to broker, skip: %v", err)
		t.Skip()
	}
	defer b.Disconnect()

	// 双触发机制
	// 1. 时间驱动：即使消息数量未达到batchSize，只要batchInterval超时，就会触发批处理。
	// 2. 数量驱动：若消息堆积速度快，当消息数量达到batchSize时，立即触发批处理（不等待batchInterval）。

	_, err := b.Subscribe(testTopic,
		RegisterHygrothermographHandler(handleHygrothermograph),
		api.HygrothermographCreator,
		broker.WithSubscribeQueueName(testGroupId),
		WithMinBytes(10e3),         // 10MB
		WithMaxBytes(10e3),         // 10MB
		WithMaxWait(3*time.Second), // 等待消息的最大时间
		WithReadLagInterval(-1),    // 禁用消费滞后统计以提高性能
		WithSubscribeBatchSize(100),
		WithSubscribeBatchInterval(5*time.Second),
	)
	assert.Nil(t, err)

	<-interrupt
}

// 测试 middleware 链的执行顺序
func Test_ChainPublishMiddleware_Order(t *testing.T) {
	var order []string

	// final task
	final := func(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
		order = append(order, "final")
		return nil
	}

	// middleware 构造器
	makeMw := func(name string) broker.PublishMiddleware {
		return func(next broker.PublishHandler) broker.PublishHandler {
			return func(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
				order = append(order, name+"-before")
				if err := next(ctx, topic, msg, opts...); err != nil {
					return err
				}
				order = append(order, name+"-after")
				return nil
			}
		}
	}

	// 假设 m1 在前（最外层），m2 在后（内层）
	m1 := makeMw("m1")
	m2 := makeMw("m2")

	chained := broker.ChainPublishMiddleware(final, []broker.PublishMiddleware{m1, m2})

	err := chained(context.Background(), "topic", broker.NewMessage("msg"))
	assert.Nil(t, err)

	// 期望顺序： m1-before, m2-before, final, m2-after, m1-after
	expected := []string{"m1-before", "m2-before", "final", "m2-after", "m1-after"}
	assert.Equal(t, expected, order)
}

// 测试 per-call 中间件与 broker 级别中间件合并执行顺序（per-call 优先）
func Test_Merge_PerCall_And_Broker_Middlewares_Order(t *testing.T) {
	var order []string

	final := func(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
		order = append(order, "final")
		return nil
	}

	makeMw := func(name string) broker.PublishMiddleware {
		return func(next broker.PublishHandler) broker.PublishHandler {
			return func(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
				order = append(order, name+"-before")
				if err := next(ctx, topic, msg, opts...); err != nil {
					return err
				}
				order = append(order, name+"-after")
				return nil
			}
		}
	}

	// 模拟 per-call 与 broker-level 中间件合并（per-call 在前）
	perCall := []broker.PublishMiddleware{makeMw("pc1"), makeMw("pc2")}
	brokerLevel := []broker.PublishMiddleware{makeMw("b1")}

	// 按实现中的合并逻辑：先 append per-call，再 append broker-level
	combined := append([]broker.PublishMiddleware{}, perCall...)
	combined = append(combined, brokerLevel...)

	chained := broker.ChainPublishMiddleware(final, combined)

	err := chained(context.Background(), "topic", broker.NewMessage("msg"))
	assert.Nil(t, err)

	// 期望顺序： pc1-before, pc2-before, b1-before, final, b1-after, pc2-after, pc1-after
	expected := []string{
		"pc1-before", "pc2-before", "b1-before",
		"final",
		"b1-after", "pc2-after", "pc1-after",
	}
	assert.Equal(t, expected, order)
}
