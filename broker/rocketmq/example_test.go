package rocketmq_test

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/broker/rocketmq"
	rocketmqOption "github.com/tx7do/kratos-transport/broker/rocketmq/option"
	api "github.com/tx7do/kratos-transport/testing/api/manual"
)

func handleHygrothermograph(_ context.Context, topic string, headers broker.Headers, msg *api.Hygrothermograph) error {
	fmt.Printf("Topic %s, Headers: %+v, Payload: %+v\n", topic, headers, msg)
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

// ExampleNewBroker 演示通过 RocketMQ（V5 客户端驱动）订阅主题并按强类型消费消息。
// 没有 Output 注释，go test 只编译不执行；实际运行需要本地 RocketMQ NameServer (127.0.0.1:9876) 及 Proxy (127.0.0.1:8081)。
func ExampleNewBroker() {
	b := rocketmq.NewBroker(
		rocketmqOption.DriverTypeV5,
		broker.WithCodec("json"),
		rocketmqOption.WithNameServer([]string{"127.0.0.1:9876"}),
		broker.WithAddress("127.0.0.1:8081"),
		rocketmqOption.WithGroupName("test_group"),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		fmt.Println(err)
		return
	}
	defer func(b broker.Broker) {
		if err := b.Disconnect(); err != nil {
			fmt.Println(err)
		}
	}(b)

	_, _ = b.Subscribe("test_topic",
		RegisterHygrothermographRawHandler(handleHygrothermograph),
		api.HygrothermographCreator,
	)
}
