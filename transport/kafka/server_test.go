package kafka

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/tx7do/kratos-transport/broker"
	"net"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"testing"
)

func TestServer(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		Address("127.0.0.1:9092"),
		Subscribe("logger.sensor.ts", "fx-group", receive),
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

func receive(_ context.Context, event broker.Event) error {
	fmt.Println("Topic: ", event.Topic(), " Payload: ", string(event.Message().Body))
	//_ = event.Ack()
	return nil
}

func TestClient(t *testing.T) {

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
