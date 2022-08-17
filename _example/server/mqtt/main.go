package main

import (
	"context"
	api "github.com/tx7do/kratos-transport/_example/api/manual"
	"log"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/transport/mqtt"
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
	log.Printf("Humidity: %.2f Temperature: %.2f\n", msg.Humidity, msg.Temperature)
	return nil
}

func main() {
	ctx := context.Background()

	mqttSrv := mqtt.NewServer(
		mqtt.WithAddress([]string{EmqxCnBroker}),
		mqtt.WithCodec(encoding.GetCodec("json")),
	)

	_ = mqttSrv.RegisterSubscriber(ctx,
		"topic/bobo/#",
		api.RegisterHygrothermographJsonHandler(handleHygrothermograph),
		api.HygrothermographCreator,
	)

	app := kratos.New(
		kratos.Name("mqtt"),
		kratos.Server(
			mqttSrv,
		),
	)
	if err := app.Run(); err != nil {
		log.Println(err)
	}
}
