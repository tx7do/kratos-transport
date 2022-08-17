package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-kratos/kratos/v2/encoding"
	api "github.com/tx7do/kratos-transport/_example/api/manual"
	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/broker/rabbitmq"
)

const (
	testBroker = "amqp://user:bitnami@127.0.0.1:5672"

	testExchange = "test_exchange"
	testQueue    = "test_queue"
	testRouting  = "test_routing_key"
)

func handleHygrothermograph(_ context.Context, topic string, headers broker.Headers, msg *api.Hygrothermograph) error {
	log.Printf("Headers: %+v, Humidity: %.2f Temperature: %.2f\n", headers, msg.Humidity, msg.Temperature)
	return nil
}

func main() {
	ctx := context.Background()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	b := rabbitmq.NewBroker(
		broker.WithOptionContext(ctx),
		broker.WithCodec(encoding.GetCodec("json")),
		broker.WithAddress(testBroker),
		rabbitmq.WithExchangeName(testExchange),
		rabbitmq.WithDurableExchange(),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		fmt.Println(err)
	}

	_, _ = b.Subscribe(testRouting,
		api.RegisterHygrothermographJsonHandler(handleHygrothermograph),
		api.HygrothermographCreator,
		broker.WithSubscribeContext(ctx),
		broker.WithQueueName(testQueue),
		// broker.WithDisableAutoAck(),
		rabbitmq.WithDurableQueue(),
	)

	<-interrupt
}
