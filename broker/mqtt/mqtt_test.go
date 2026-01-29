package mqtt

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
	"github.com/stretchr/testify/assert"

	"github.com/tx7do/kratos-transport/broker"
	api "github.com/tx7do/kratos-transport/testing/api/manual"
)

const (
	EmqxBroker        = "tcp://broker.emqx.io:1883"
	EmqxCnBroker      = "tcp://broker-cn.emqx.io:1883"
	EclipseBroker     = "tcp://mqtt.eclipseprojects.io:1883"
	MosquittoBroker   = "tcp://test.mosquitto.org:1883"
	HiveMQBroker      = "tcp://broker.hivemq.com:1883"
	LocalEmxqBroker   = "tcp://127.0.0.1:1883"
	LocalRabbitBroker = "tcp://user:bitnami@127.0.0.1:1883"
)

func handleHygrothermograph(_ context.Context, topic string, headers broker.Headers, msg *api.Hygrothermograph) error {
	LogInfof("Topic %s, Headers: %+v, Payload: %+v\n", topic, headers, msg)
	return nil
}

func Test_Publish_WithRawData(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()
	var err error

	b := NewBroker(
		broker.WithOptionContext(ctx),
		broker.WithAddress(EmqxBroker),
	)

	if err = b.Init(); err != nil {
		t.Logf("init broker failed, skip: %v", err)
		t.Skip()
	}

	if err = b.Connect(); err != nil {
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
		err = b.Publish(ctx, "topic/bobo/1", broker.NewMessage(buf))
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

	var err error
	ctx := context.Background()

	b := NewBroker(
		broker.WithAddress(EmqxBroker),
		WithErrorLogger(),
		WithCriticalLogger(),
		//WithWarnLogger(),
		//WithDebugLogger(),
	)

	if err = b.Init(); err != nil {
		t.Logf("init broker failed, skip: %v", err)
		t.Skip()
	}

	if err = b.Connect(); err != nil {
		t.Logf("cant connect to broker, skip: %v", err)
		t.Skip()
	}
	defer b.Disconnect()

	_, err = b.Subscribe("topic/bobo/#",
		RegisterHygrothermographRawHandler(handleHygrothermograph),
		nil,
		broker.WithSubscribeContext(ctx),
	)
	assert.Nil(t, err)

	<-interrupt
}

func Test_Publish_WithJsonCodec(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()
	var err error

	b := NewBroker(
		broker.WithOptionContext(ctx),
		broker.WithAddress(LocalRabbitBroker),
		broker.WithCodec("json"),
	)

	if err = b.Init(); err != nil {
		t.Logf("init broker failed, skip: %v", err)
		t.Skip()
	}

	if err = b.Connect(); err != nil {
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
		err = b.Publish(ctx, "topic/bobo/1", broker.NewMessage(msg))
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

	ctx := context.Background()
	var err error

	b := NewBroker(
		broker.WithAddress(EmqxBroker),
		broker.WithCodec("json"),
	)

	if err = b.Init(); err != nil {
		t.Logf("init broker failed, skip: %v", err)
		t.Skip()
	}

	if err = b.Connect(); err != nil {
		t.Logf("cant connect to broker, skip: %v", err)
		t.Skip()
	}
	defer b.Disconnect()

	_, err = b.Subscribe("topic/bobo/#",
		RegisterHygrothermographRawHandler(handleHygrothermograph),
		api.HygrothermographCreator,
		broker.WithSubscribeContext(ctx),
	)
	assert.Nil(t, err)

	<-interrupt
}

type HygrothermographHandler func(_ context.Context, topic string, headers broker.Headers, msg *api.Hygrothermograph) error

func RegisterHygrothermographRawHandler(fnc HygrothermographHandler) broker.Handler {
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
