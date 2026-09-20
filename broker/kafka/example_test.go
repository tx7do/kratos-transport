package kafka_test

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/broker/kafka"
	api "github.com/tx7do/kratos-transport/testing/api/manual"
)

const (
	testBrokers = "localhost:9092"
	testGroupId = "a-group"
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

// ExampleNewBroker 演示订阅 Kafka 主题并按消费组消费强类型消息。
// 没有 Output 注释，go test 只编译不执行；实际运行需要本地 Kafka (localhost:9092)。
func ExampleNewBroker() {
	b := kafka.NewBroker(
		broker.WithAddress(testBrokers),
		broker.WithCodec("json"),
	)

	_ = b.Init()

	_, err := b.Subscribe("test_topic",
		RegisterHygrothermographRawHandler(handleHygrothermograph),
		api.HygrothermographCreator,
		broker.WithSubscribeQueueName(testGroupId),
	)
	if err != nil {
		log.Error(err)
	}
}
