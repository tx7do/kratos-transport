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
	"github.com/tx7do/kratos-transport/broker/rocketmq"
)

const (
	// Name Server address
	testBroker = "127.0.0.1:9876"

	testTopic     = "test_topic"
	testGroupName = "CID_ONSAPI_OWNER"
)

func handleHygrothermograph(_ context.Context, topic string, headers broker.Headers, msg *api.Hygrothermograph) error {
	log.Printf("Humidity: %.2f Temperature: %.2f\n", msg.Humidity, msg.Temperature)
	return nil
}

func TestServer(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		WithNameServer([]string{testBroker}),
		//WithNameServerDomain("http://nsaddr.rmq.cloud.tencent.com"),
		WithCodec(encoding.GetCodec("json")),
	)

	err := srv.RegisterSubscriber(ctx, testTopic, testGroupName,
		api.RegisterHygrothermographJsonHandler(handleHygrothermograph),
		func() broker.Any {
			return &api.Hygrothermograph{}
		})
	assert.Nil(t, err)

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

	b := rocketmq.NewBroker(
		broker.WithOptionContext(ctx),
		broker.WithCodec(encoding.GetCodec("json")),
		rocketmq.WithEnableTrace(),
		rocketmq.WithNameServer([]string{testBroker}),
		//rocketmq.WithNameServerDomain(testBroker),
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

func TestAliyunServer(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()
	endpoint := ""
	accessKey := ""
	secretKey := ""
	instanceId := ""
	topicName := ""
	groupName := "GID_DEFAULT"

	srv := NewServer(
		WithAliyunHttpSupport(),
		WithCodec(encoding.GetCodec("json")),
		WithEnableTrace(),
		WithNameServerDomain(endpoint),
		WithCredentials(accessKey, secretKey, ""),
		WithInstanceName(instanceId),
		WithGroupName(groupName),
	)

	err := srv.RegisterSubscriber(ctx, topicName, groupName,
		api.RegisterHygrothermographJsonHandler(handleHygrothermograph),
		func() broker.Any {
			return &api.Hygrothermograph{}
		})
	assert.Nil(t, err)

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

func TestAliyunClient(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()
	endpoint := ""
	accessKey := ""
	secretKey := ""
	instanceId := ""
	topicName := ""

	b := rocketmq.NewBroker(
		broker.WithOptionContext(ctx),
		broker.WithCodec(encoding.GetCodec("json")),
		rocketmq.WithAliyunHttpSupport(),
		rocketmq.WithEnableTrace(),
		rocketmq.WithNameServerDomain(endpoint),
		rocketmq.WithAccessKey(accessKey),
		rocketmq.WithSecretKey(secretKey),
		rocketmq.WithInstanceName(instanceId),
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
