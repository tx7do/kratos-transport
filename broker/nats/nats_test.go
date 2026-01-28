package nats

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

	natsGo "github.com/nats-io/nats.go"

	"github.com/stretchr/testify/assert"

	"github.com/tx7do/kratos-transport/broker"
	api "github.com/tx7do/kratos-transport/testing/api/manual"
	"github.com/tx7do/kratos-transport/tracing"
)

var addrTestCases = []struct {
	name        string
	description string
	addrs       map[string]string // expected address : set address
}{
	{
		"commonOpts",
		"set common addresses through a common.Option in constructor",
		map[string]string{
			"nats://192.168.10.1:5222": "192.168.10.1:5222",
			"nats://10.20.10.0:4222":   "10.20.10.0:4222"},
	},
	{
		"commonInit",
		"set common addresses through a common.Option in common.Init()",
		map[string]string{
			"nats://192.168.10.1:5222": "192.168.10.1:5222",
			"nats://10.20.10.0:4222":   "10.20.10.0:4222"},
	},
	{
		"natsOpts",
		"set common addresses through the nats.Option in constructor",
		map[string]string{
			"nats://192.168.10.1:5222": "192.168.10.1:5222",
			"nats://10.20.10.0:4222":   "10.20.10.0:4222"},
	},
	{
		"default",
		"check if default Address is set correctly",
		map[string]string{
			"nats://127.0.0.1:4222": "",
		},
	},
}

// TestInitAddrs tests issue #100. Ensures that if the addrs is set by an option in init it will be used.
func TestInitAddrs(t *testing.T) {

	for _, tc := range addrTestCases {
		t.Run(fmt.Sprintf("%s: %s", tc.name, tc.description), func(t *testing.T) {

			var br broker.Broker
			var addrs []string

			for _, addr := range tc.addrs {
				addrs = append(addrs, addr)
			}

			switch tc.name {
			case "commonOpts":
				// we know that there are just two addrs in the dict
				br = NewBroker(broker.WithAddress(addrs[0], addrs[1]))
				_ = br.Init()
			case "commonInit":
				br = NewBroker()
				// we know that there are just two addrs in the dict
				_ = br.Init(broker.WithAddress(addrs[0], addrs[1]))
			case "natsOpts":
				nopts := natsGo.GetDefaultOptions()
				nopts.Servers = addrs
				br = NewBroker(Options(nopts))
				_ = br.Init()
			case "default":
				br = NewBroker()
				_ = br.Init()
			}

			b, ok := br.(*natsBroker)
			if !ok {
				t.Fatal("Expected common to be of types *b")
			}
			// check if the same amount of addrs we set has actually been set, default
			// have only 1 address nats://127.0.0.1:4222 (current nats code) or
			// nats://localhost:4222 (older code version)
			if len(b.options.Addrs) != len(tc.addrs) && tc.name != "default" {
				t.Errorf("Expected Addr count = %d, Actual Addr count = %d",
					len(b.options.Addrs), len(tc.addrs))
			}

			for _, addr := range b.options.Addrs {
				_, ok := tc.addrs[addr]
				if !ok {
					t.Errorf("Expected '%s' has not been set", addr)
				}
			}
		})
	}
}

const (
	localBroker = "nats://127.0.0.1:4222"
	testTopic   = "test_topic"
)

// RegisterHygrothermographResponseJsonHandler 将一个返回 (any, error) 的处理函数适配为 broker.Handler，
// 并在收到消息后将函数返回序列化并通过 NATS Respond 回应。
func RegisterHygrothermographResponseJsonHandler(fnc func(ctx context.Context, topic string, headers broker.Headers, msg *api.Hygrothermograph) (any, error)) broker.Handler {
	return func(ctx context.Context, event broker.Event) error {
		switch t := event.Message().Body.(type) {
		case *api.Hygrothermograph:
			res, err := fnc(ctx, event.Topic(), event.Message().Headers, t)
			if err != nil {
				return err
			}
			rawMsg, _ := json.Marshal(res)
			if m, ok := event.Message().Msg.(*natsGo.Msg); ok {
				_ = m.Respond(rawMsg)
			}
		default:
			return fmt.Errorf("unsupported type: %T", t)
		}
		return nil
	}
}

// RegisterHygrothermographHandler 将一个不返回响应的业务函数适配为 broker.Handler。
func RegisterHygrothermographHandler(f func(ctx context.Context, topic string, headers broker.Headers, msg *api.Hygrothermograph) error) broker.Handler {
	return func(ctx context.Context, event broker.Event) error {
		switch t := event.Message().Body.(type) {
		case *api.Hygrothermograph:
			return f(ctx, event.Topic(), event.Message().Headers, t)
		default:
			return fmt.Errorf("unsupported type: %T", t)
		}
	}
}

func responseHandleHygrothermograph(_ context.Context, topic string, headers broker.Headers, msg *api.Hygrothermograph) (any, error) {
	LogInfof("Topic %s, Headers: %+v, Payload: %+v\n", topic, headers, msg)
	return msg, nil
}

func handleHygrothermograph(_ context.Context, topic string, headers broker.Headers, msg *api.Hygrothermograph) error {
	LogInfof("Topic %s, Headers: %+v, Payload: %+v\n", topic, headers, msg)
	return nil
}

func Test_Publish_WithRawData(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	b := NewBroker(
		broker.WithAddress(localBroker),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		t.Logf("cant connect to broker, skip: %v", err)
		t.Skip()
	}
	defer b.Disconnect()

	ctx := context.Background()

	var msg api.Hygrothermograph
	const count = 10
	for i := 0; i < count; i++ {
		startTime := time.Now()
		msg.Humidity = float64(rand.Intn(100))
		msg.Temperature = float64(rand.Intn(100))
		buf, _ := json.Marshal(&msg)
		err := b.Publish(ctx, testTopic, buf)
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
		broker.WithAddress(localBroker),
	)
	defer b.Disconnect()

	_ = b.Connect()

	_, err := b.Subscribe(testTopic,
		RegisterHygrothermographHandler(handleHygrothermograph),
		nil,
	)
	assert.Nil(t, err)

	<-interrupt
}

func Test_Publish_WithJsonCodec(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	b := NewBroker(
		broker.WithAddress(localBroker),
		broker.WithCodec("json"),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		t.Logf("cant connect to broker, skip: %v", err)
		t.Skip()
	}
	defer b.Disconnect()

	ctx := context.Background()

	var msg api.Hygrothermograph
	const count = 10
	for i := 0; i < count; i++ {
		startTime := time.Now()
		msg.Humidity = float64(rand.Intn(100))
		msg.Temperature = float64(rand.Intn(100))
		err := b.Publish(ctx, testTopic, msg)
		assert.Nil(t, err)
		elapsedTime := time.Since(startTime) / time.Millisecond
		fmt.Printf("Publish %d, elapsed time: %dms, Humidity: %.2f Temperature: %.2f\n",
			i, elapsedTime, msg.Humidity, msg.Temperature)
	}

	fmt.Printf("total send %d messages\n", count)

	<-interrupt
}

func Test_Subscribe_WithJsonCodec(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	b := NewBroker(
		broker.WithAddress(localBroker),
		broker.WithCodec("json"),
	)
	defer b.Disconnect()

	_ = b.Connect()

	_, err := b.Subscribe(testTopic,
		RegisterHygrothermographHandler(handleHygrothermograph),
		api.HygrothermographCreator,
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

	b := NewBroker(
		broker.WithAddress(localBroker),
		broker.WithCodec("json"),
		createTracerProvider("otlp-grpc", "publish_tracer_tester"),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		t.Logf("cant connect to broker, skip: %v", err)
		t.Skip()
	}
	defer b.Disconnect()

	ctx := context.Background()

	var msg api.Hygrothermograph
	const count = 10
	for i := 0; i < count; i++ {
		startTime := time.Now()
		msg.Humidity = float64(rand.Intn(100))
		msg.Temperature = float64(rand.Intn(100))
		err := b.Publish(ctx, testTopic, msg)
		assert.Nil(t, err)
		elapsedTime := time.Since(startTime) / time.Millisecond
		fmt.Printf("Publish %d, elapsed time: %dms, Humidity: %.2f Temperature: %.2f\n",
			i, elapsedTime, msg.Humidity, msg.Temperature)
	}

	fmt.Printf("total send %d messages\n", count)

	<-interrupt
}

func Test_Subscribe_WithTracer(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	b := NewBroker(
		broker.WithAddress(localBroker),
		broker.WithCodec("json"),
		createTracerProvider("otlp-grpc", "subscribe_tracer_tester"),
	)
	defer b.Disconnect()

	_ = b.Connect()

	_, err := b.Subscribe(testTopic,
		RegisterHygrothermographHandler(handleHygrothermograph),
		api.HygrothermographCreator,
	)
	assert.Nil(t, err)

	<-interrupt
}

func Test_Request_WithTracer(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	b := NewBroker(
		broker.WithAddress(localBroker),
		broker.WithCodec("json"),
		createTracerProvider("otlp-grpc", "request_tracer_tester"),
	)
	_ = b.Init()

	if err := b.Connect(); err != nil {
		t.Logf("cant connect to broker, skip: %v", err)
		t.Skip()
	}

	defer b.Disconnect()
	ctx := context.Background()

	var msg api.Hygrothermograph

	const count = 10
	for i := 0; i < count; i++ {
		startTime := time.Now()
		msg.Humidity = float64(rand.Intn(100))
		msg.Temperature = float64(rand.Intn(100))

		reply, err := b.Request(ctx, testTopic, msg, WithRequestTimeout(time.Second*2))
		assert.Nil(t, err)

		elapsedTime := time.Since(startTime) / time.Millisecond

		natsMsg := reply.(*natsGo.Msg)
		res := api.Hygrothermograph{}
		err = json.Unmarshal(natsMsg.Data, &res)
		assert.Nil(t, err)

		fmt.Printf("Response %d, elapsed time: %dms, Humidity: %.2f Temperature: %.2f\n",
			i, elapsedTime, res.Humidity, res.Temperature)
	}

	fmt.Printf("total send %d messages\n", count)

	<-interrupt
}

func Test_ResponseSubscribe_WithTracer(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	b := NewBroker(
		broker.WithAddress(localBroker),
		broker.WithCodec("json"),
		createTracerProvider("otlp-grpc", "responseSubscribe_tracer_tester"),
	)

	defer b.Disconnect()
	_ = b.Connect()

	_, err := b.Subscribe(
		testTopic,
		RegisterHygrothermographResponseJsonHandler(responseHandleHygrothermograph),
		api.HygrothermographCreator,
	)

	assert.Nil(t, err)

	<-interrupt
}
