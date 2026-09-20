package mqtt_test

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/broker/mqtt"
	api "github.com/tx7do/kratos-transport/testing/api/manual"
)

func handleHygrothermograph(_ context.Context, topic string, headers broker.Headers, msg *api.Hygrothermograph) error {
	log.Infof("Topic %s, Headers: %+v, Payload: %+v\n", topic, headers, msg)
	return nil
}

type HygrothermographHandler func(_ context.Context, topic string, headers broker.Headers, msg *api.Hygrothermograph) error

// RegisterHygrothermographRawHandler 把裸字节流解码成强类型消息后转交给业务处理函数。
func RegisterHygrothermographRawHandler(fnc HygrothermographHandler) broker.Handler {
	return func(ctx context.Context, event broker.Event) error {
		var msg api.Hygrothermograph

		switch t := event.Message().Body.(type) {
		case []byte:
			if err := json.Unmarshal(t, &msg); err != nil {
				return fmt.Errorf("json unmarshal failed: %w", err)
			}
		case string:
			if err := json.Unmarshal([]byte(t), &msg); err != nil {
				return fmt.Errorf("json unmarshal failed: %w", err)
			}
		default:
			return fmt.Errorf("unsupported type: %T", t)
		}

		return fnc(ctx, event.Topic(), event.Message().Headers, &msg)
	}
}

// ExampleNewBroker 演示连接 MQTT broker 并通配订阅主题。
// 共享订阅可把 topic 换成 "$share/g1/topic/bobo/#" 或 "$queue/topic/bobo/#"。
// 没有 Output 注释，go test 只编译不执行；实际运行需要本地 EMQX (tcp://127.0.0.1:1883)。
func ExampleNewBroker() {
	b := mqtt.NewBroker(
		broker.WithCodec("json"),
		broker.WithAddress("tcp://127.0.0.1:1883"),
		mqtt.WithCleanSession(false),
		mqtt.WithAuth("user", "bitnami"),
		mqtt.WithClientId("test-client-2"),
	)

	if err := b.Connect(); err != nil {
		log.Error(err)
		return
	}
	defer func(b broker.Broker) {
		if err := b.Disconnect(); err != nil {
			log.Error(err)
		}
	}(b)

	_, err := b.Subscribe("topic/bobo/#",
		RegisterHygrothermographRawHandler(handleHygrothermograph),
		api.HygrothermographCreator,
	)
	if err != nil {
		log.Error(err)
	}
}
