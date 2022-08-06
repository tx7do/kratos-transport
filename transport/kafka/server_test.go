package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"testing"

	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/stretchr/testify/assert"
	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/broker/kafka"
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
	log.Printf("Humidity: %.2f Temperature: %.2f\n", msg.Humidity, msg.Temperature)
	return nil
}

func TestServer(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		WithAddress([]string{testBrokers}),
		WithCodec(encoding.GetCodec("json")),
	)

	_ = srv.RegisterSubscriber(ctx,
		testTopic, testGroupId, false,
		registerHygrothermographJsonHandler(),
		func() broker.Any {
			return &Hygrothermograph{}
		})

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

	ctx := context.Background()

	b := kafka.NewBroker(
		broker.OptionContext(ctx),
		broker.Addrs(testBrokers),
		broker.Codec(encoding.GetCodec("json")),
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
