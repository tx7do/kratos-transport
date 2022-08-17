package rocketmq

import (
	"context"
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
	testBroker    = "127.0.0.1:9876"
	testTopic     = "test_topic"
	testGroupName = "CID_ONSAPI_OWNER"
)

func handleHygrothermograph(_ context.Context, _ string, _ broker.Headers, msg *api.Hygrothermograph) error {
	log.Printf("Humidity: %.2f Temperature: %.2f\n", msg.Humidity, msg.Temperature)
	return nil
}

func TestSubscribe(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	b := NewBroker(
		broker.WithOptionContext(ctx),
		broker.WithCodec(encoding.GetCodec("json")),
		WithNameServer([]string{testBroker}),
		//WithNameServerDomain(testBroker),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		t.Logf("cant connect to broker, skip: %v", err)
		t.Skip()
	}

	_, err := b.Subscribe(testTopic,
		api.RegisterHygrothermographJsonHandler(handleHygrothermograph),
		api.HygrothermographCreator,
		broker.WithSubscribeContext(ctx),
		broker.WithQueueName(testGroupName),
	)
	assert.Nil(t, err)

	<-interrupt
}

func TestPublish(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	b := NewBroker(
		broker.WithOptionContext(ctx),
		broker.WithCodec(encoding.GetCodec("json")),
		WithEnableTrace(),
		WithNameServer([]string{testBroker}),
		//WithNameServerDomain(testBroker),
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
		err := b.Publish(testTopic, msg)
		assert.Nil(t, err)
		elapsedTime := time.Since(startTime) / time.Millisecond
		fmt.Printf("Publish %d, elapsed time: %dms, Humidity: %.2f Temperature: %.2f\n",
			i, elapsedTime, msg.Humidity, msg.Temperature)
	}

	fmt.Printf("total send %d messages\n", count)

	<-interrupt
}

func TestAliyunPublish(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()
	endpoint := ""
	accessKey := ""
	secretKey := ""
	instanceId := ""
	topicName := ""

	b := NewBroker(
		broker.WithOptionContext(ctx),
		broker.WithCodec(encoding.GetCodec("json")),
		WithAliyunHttpSupport(),
		WithEnableTrace(),
		WithNameServerDomain(endpoint),
		WithAccessKey(accessKey),
		WithSecretKey(secretKey),
		WithInstanceName(instanceId),
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
		err := b.Publish(topicName, msg)
		assert.Nil(t, err)
		elapsedTime := time.Since(startTime) / time.Millisecond
		fmt.Printf("Publish %d, elapsed time: %dms, Humidity: %.2f Temperature: %.2f\n",
			i, elapsedTime, msg.Humidity, msg.Temperature)
	}

	fmt.Printf("total send %d messages\n", count)

	<-interrupt
}

func TestAliyunSubscribe(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()
	endpoint := ""
	accessKey := ""
	secretKey := ""
	instanceId := ""
	topicName := ""
	groupName := "GID_DEFAULT"

	b := NewBroker(
		broker.WithOptionContext(ctx),
		broker.WithCodec(encoding.GetCodec("json")),
		WithAliyunHttpSupport(),
		WithEnableTrace(),
		WithNameServerDomain(endpoint),
		WithAccessKey(accessKey),
		WithSecretKey(secretKey),
		WithInstanceName(instanceId),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		t.Logf("cant connect to broker, skip: %v", err)
		t.Skip()
	}

	_, err := b.Subscribe(topicName,
		api.RegisterHygrothermographJsonHandler(handleHygrothermograph),
		api.HygrothermographCreator,
		broker.WithQueueName(groupName),
	)
	assert.Nil(t, err)

	<-interrupt
}
