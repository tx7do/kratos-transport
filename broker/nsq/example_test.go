package nsq_test

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/broker/nsq"
	api "github.com/tx7do/kratos-transport/testing/api/manual"
)

func handleHygrothermograph(_ context.Context, topic string, headers broker.Headers, msg *api.Hygrothermograph) error {
	nsq.LogInfof("Topic %s, Headers: %+v, Payload: %+v\n", topic, headers, msg)
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

// ExampleNewBroker 演示通过 NSQ 驱动订阅主题（消费通道）并按强类型消费消息。
// 没有 Output 注释，go test 只编译不执行；实际运行需要本地 NSQ (127.0.0.1:4150)。
func ExampleNewBroker() {
	b := nsq.NewBroker(
		broker.WithCodec("json"),
		broker.WithAddress("127.0.0.1:4150"),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		nsq.LogError(err)
		return
	}
	defer func(b broker.Broker) {
		if err := b.Disconnect(); err != nil {
			nsq.LogError(err)
		}
	}(b)

	_, _ = b.Subscribe("test_topic",
		RegisterHygrothermographRawHandler(handleHygrothermograph),
		api.HygrothermographCreator,
		broker.WithSubscribeQueueName("test_channel"),
	)
}
