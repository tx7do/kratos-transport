package kafka

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	api "github.com/tx7do/kratos-transport/testing/api/manual"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/broker/kafka"
	"github.com/tx7do/kratos-transport/tracing"
)

const (
	testBrokers = "localhost:9092"
	testTopic   = "logger.sensor.ts"
	testGroupId = "fx-group"
)

func handleHygrothermograph(_ context.Context, topic string, headers broker.Headers, msg *api.Hygrothermograph) error {
	LogInfof("Topic %s, Headers: %+v, Payload: %+v\n", topic, headers, msg)
	return nil
}

func loggingMiddlerware(next broker.Handler) broker.Handler {
	return func(ctx context.Context, evt broker.Event) error {
		err := next(ctx, evt)
		if err != nil {
			LogErrorf("err is %+v", err)
		}
		return nil
	}
}

func recoveryMiddlerware(next broker.Handler) broker.Handler {
	return func(ctx context.Context, evt broker.Event) error {
		defer func() {
			if rerr := recover(); rerr != nil {
				LogError(rerr)
			}
		}()
		LogInfo("recover middleware")
		return next(ctx, evt)
	}
}

func TestServer(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		WithAddress([]string{testBrokers}),
		WithCodec("json"),
		WithBrokerOptions(
			broker.WithSubscriberMiddlewares(
				loggingMiddlerware,
				recoveryMiddlerware,
			),
		),
	)

	_ = RegisterSubscriber(srv, ctx,
		testTopic, testGroupId, false,
		handleHygrothermograph,
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

func TestClient(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	b := kafka.NewBroker(
		broker.WithAddress(testBrokers),
		broker.WithCodec("json"),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		t.Logf("cant connect to broker, skip: %v", err)
		t.Skip()
	}
	defer b.Disconnect()

	_, err := broker.Subscribe(b,
		testTopic,
		handleHygrothermograph,
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

func TestServerWithTracer(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		WithAddress([]string{testBrokers}),
		WithCodec("json"),
		WithBrokerOptions(createTracerProvider("otlp-grpc", "tracer_tester")),
	)

	_ = RegisterSubscriber(srv,
		ctx,
		testTopic, testGroupId, false,
		handleHygrothermograph,
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

func TestParseIP(t *testing.T) {
	IP1 := "www.baidu.com"
	IP2 := "127.0.0.1"
	IP3 := "127.0.0.1:8080"
	parsedIP1 := net.ParseIP(IP1)
	parsedIP2 := net.ParseIP(IP2)
	parsedIP3 := net.ParseIP(IP3)
	fmt.Println("net.ParseIP: ", parsedIP1, parsedIP2, parsedIP3)
}

func TestParseUrl(t *testing.T) {
	IP1 := "www.baidu.com"
	IP2 := "127.0.0.1"
	IP3 := "127.0.0.1:8080"
	IP4 := "tcp://127.0.0.1:8080"

	parsedIP1, err := url.Parse(IP1)
	assert.Nil(t, err)
	assert.Equal(t, parsedIP1.Path, "www.baidu.com")

	parsedIP2, err := url.Parse(IP2)
	assert.Nil(t, err)
	assert.Equal(t, parsedIP2.Path, "127.0.0.1")

	parsedIP3, err := url.Parse(IP3)
	assert.NotNil(t, err)
	assert.Nil(t, parsedIP3)

	parsedIP4, err := url.Parse(IP4)
	assert.Nil(t, err)
	assert.Equal(t, parsedIP4.Host, "127.0.0.1:8080")
}
