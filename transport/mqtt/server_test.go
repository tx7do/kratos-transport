package mqtt

import (
	"context"
	"fmt"
	"github.com/tx7do/kratos-transport/broker"
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

func TestServer(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		Address(EmqxCnBroker),
		Subscribe("topic/bobo/#", receive),
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

func receive(event broker.Event) error {
	fmt.Println("Topic: ", event.Topic(), " Payload: ", string(event.Message().Body))
	return nil
}

func TestClient(t *testing.T) {

}
