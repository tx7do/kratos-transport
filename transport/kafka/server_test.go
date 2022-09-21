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

	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/stretchr/testify/assert"

	api "github.com/tx7do/kratos-transport/_example/api/manual"
	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/broker/kafka"
)

const (
	testBrokers = "localhost:9092"
	testTopic   = "logger.sensor.ts"
	testGroupId = "fx-group"
)

func handleHygrothermograph(_ context.Context, topic string, headers broker.Headers, msg *api.Hygrothermograph) error {
	log.Infof("Topic %s, Headers: %+v, Payload: %+v\n", topic, headers, msg)
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
		api.RegisterHygrothermographJsonHandler(handleHygrothermograph),
		api.HygrothermographCreator,
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

	ctx := context.Background()

	b := kafka.NewBroker(
		broker.WithOptionContext(ctx),
		broker.WithAddress(testBrokers),
		broker.WithCodec(encoding.GetCodec("json")),
	)

	_, err := b.Subscribe(testTopic,
		api.RegisterHygrothermographJsonHandler(handleHygrothermograph),
		api.HygrothermographCreator,
		broker.WithSubscribeContext(ctx),
		broker.WithQueueName(testGroupId),
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
