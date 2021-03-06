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

	"github.com/stretchr/testify/assert"
	"github.com/tx7do/kratos-transport/broker"
	jsonCodec "github.com/tx7do/kratos-transport/codec/json"
)

const (
	testBrokers = "localhost:9092"
	testTopic   = "logger.sensor.ts"
	testGroupId = "fx-group"
)

type Hygrothermograph struct {
	Humidity    float64 `json:"humidity"`
	Temperature float64 `json:"temperature"`
}

type Headers broker.Headers

func TestCodec(t *testing.T) {
	codec := jsonCodec.Marshaler{}

	var msg Hygrothermograph
	msg.Humidity = 100
	msg.Temperature = 200

	buf, err := codec.Marshal(msg)
	assert.Nil(t, err)

	var msg1 Hygrothermograph
	var body interface{}
	body = &msg1
	err = codec.Unmarshal(buf, body)
	assert.Nil(t, err)
	msg2, _ := body.(*Hygrothermograph)
	assert.Equal(t, msg, msg2)
}

func registerHygrothermographRawHandler() broker.Handler {
	return func(ctx context.Context, event broker.Event) error {
		var msg Hygrothermograph

		switch t := event.Message().Body.(type) {
		case []byte:
			if err := json.Unmarshal(t, &msg); err != nil {
				return err
			}
		case string:
			if err := json.Unmarshal([]byte(t), &msg); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported type: %T", t)
		}

		if err := handleHygrothermograph(ctx, event.Topic(), event.Message().Headers, &msg); err != nil {
			return err
		}

		return nil
	}
}

func registerHygrothermographJsonHandler() broker.Handler {
	return func(ctx context.Context, event broker.Event) error {
		switch t := event.Message().Body.(type) {
		case *Hygrothermograph:
			if err := handleHygrothermograph(ctx, event.Topic(), event.Message().Headers, t); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported type: %T", t)
		}
		return nil
	}
}

func handleHygrothermograph(_ context.Context, topic string, headers broker.Headers, msg *Hygrothermograph) error {
	log.Printf("Headers: %+v, Humidity: %.2f Temperature: %.2f\n", headers, msg.Humidity, msg.Temperature)
	return nil
}

func Test_Publish_WithRawData(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	b := NewBroker(
		broker.OptionContext(ctx),
		broker.Addrs(testBrokers),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		t.Logf("cant connect to broker, skip: %v", err)
		t.Skip()
	}

	var msg Hygrothermograph
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
		broker.OptionContext(ctx),
		broker.Addrs(testBrokers),
	)

	_, err := b.Subscribe(testTopic,
		registerHygrothermographRawHandler(),
		nil,
		broker.SubscribeContext(ctx),
		broker.Queue(testGroupId),
	)
	assert.Nil(t, err)

	<-interrupt
}

func Test_Publish_WithJsonCodec(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	b := NewBroker(
		broker.OptionContext(ctx),
		broker.Addrs(testBrokers),
		broker.Codec(jsonCodec.Marshaler{}),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		t.Logf("cant connect to broker, skip: %v", err)
		t.Skip()
	}

	var headers map[string]interface{}
	headers = make(map[string]interface{})
	headers["version"] = "1.0.0"

	var msg Hygrothermograph
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
		broker.OptionContext(ctx),
		broker.Addrs(testBrokers),
		broker.Codec(jsonCodec.Marshaler{}),
	)

	_, err := b.Subscribe(testTopic,
		registerHygrothermographJsonHandler(),
		func() broker.Any {
			return &Hygrothermograph{}
		},
		broker.SubscribeContext(ctx),
		broker.Queue(testGroupId),
	)
	assert.Nil(t, err)

	<-interrupt
}
