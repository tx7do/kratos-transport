package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/stretchr/testify/assert"
	api "github.com/tx7do/kratos-transport/_example/api/manual"
	"github.com/tx7do/kratos-transport/broker"
)

const (
	testBrokers = "localhost:9092"
	testTopic   = "logger.sensor.ts"
	testGroupId = "fx-group"
)

func handleHygrothermograph(_ context.Context, topic string, headers broker.Headers, msg *api.Hygrothermograph) error {
	log.Printf("Headers: %+v, Humidity: %.2f Temperature: %.2f\n", headers, msg.Humidity, msg.Temperature)
	return nil
}

func Test_Publish_WithRawData(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	b := NewBroker(
		broker.WithOptionContext(ctx),
		broker.WithAddress(testBrokers),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		t.Logf("cant connect to broker, skip: %v", err)
		t.Skip()
	}

	var msg api.Hygrothermograph
	const count = 10
	for i := 0; i < count; i++ {
		startTime := time.Now()
		msg.Humidity = float64(rand.Intn(100))
		msg.Temperature = float64(rand.Intn(100))
		buf, _ := json.Marshal(&msg)
		err := b.Publish(testTopic, buf)
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

	ctx := context.Background()

	b := NewBroker(
		broker.WithOptionContext(ctx),
		broker.WithAddress(testBrokers),
	)

	_, err := b.Subscribe(testTopic,
		api.RegisterHygrothermographRawHandler(handleHygrothermograph),
		nil,
		broker.WithSubscribeContext(ctx),
		broker.WithQueueName(testGroupId),
	)
	assert.Nil(t, err)

	<-interrupt
}

func Test_Publish_WithJsonCodec(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	b := NewBroker(
		broker.WithOptionContext(ctx),
		broker.WithAddress(testBrokers),
		broker.WithCodec(encoding.GetCodec("json")),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		t.Logf("cant connect to broker, skip: %v", err)
		t.Skip()
	}

	var headers map[string]interface{}
	headers = make(map[string]interface{})
	headers["version"] = "1.0.0"

	var msg api.Hygrothermograph
	const count = 10
	for i := 0; i < count; i++ {
		startTime := time.Now()
		headers["trace_id"] = i
		msg.Humidity = float64(rand.Intn(100))
		msg.Temperature = float64(rand.Intn(100))
		err := b.Publish(testTopic, msg, WithHeaders(headers))
		assert.Nil(t, err)
		elapsedTime := time.Since(startTime) / time.Millisecond
		log.Printf("Publish %d, elapsed time: %dms, Humidity: %.2f Temperature: %.2f\n",
			i, elapsedTime, msg.Humidity, msg.Temperature)
	}

	log.Printf("total send %d messages\n", count)

	<-interrupt
}

func Test_Subscribe_WithJsonCodec(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	b := NewBroker(
		broker.WithOptionContext(ctx),
		broker.WithAddress(testBrokers),
		broker.WithCodec(encoding.GetCodec("json")),
	)

	_, err := b.Subscribe(testTopic,
		api.RegisterHygrothermographJsonHandler(handleHygrothermograph),
		func() broker.Any {
			return &api.Hygrothermograph{}
		},
		broker.WithSubscribeContext(ctx),
		broker.WithQueueName(testGroupId),
	)
	assert.Nil(t, err)

	<-interrupt
}
