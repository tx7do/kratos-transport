package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/tx7do/kratos-transport/broker"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"testing"
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

func TestSubscribe(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	b := NewBroker(
		broker.Addrs(LocalRabbitBroker),
		broker.OptionContext(ctx),
	)
	defer b.Disconnect()

	_ = b.Connect()

	_, err := b.Subscribe("topic/bobo/#", receive,
		broker.SubscribeContext(ctx),
	)
	assert.Nil(t, err)

	<-interrupt
}

func receive(_ context.Context, event broker.Event) error {
	fmt.Println("Topic: ", event.Topic(), " Payload: ", string(event.Message().Body))
	//_ = event.Ack()
	return nil
}

func TestPublish(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	b := NewBroker(
		broker.Addrs(EmqxCnBroker),
		broker.OptionContext(ctx),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		t.Logf("cant conect to broker, skip: %v", err)
		t.Skip()
	}

	type Hygrothermograph struct {
		Humidity    float64 `json:"humidity"`
		Temperature float64 `json:"temperature"`
	}

	var data Hygrothermograph
	data.Humidity = float64(rand.Intn(100))
	data.Temperature = float64(rand.Intn(100))

	buf, _ := json.Marshal(&data)

	var msg broker.Message
	msg.Body = buf
	_ = b.Publish("topic/bobo/1", &msg)

	<-interrupt
}
