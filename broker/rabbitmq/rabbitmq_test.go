package rabbitmq_test

import (
	"context"
	"testing"
)

type Example struct{}

func (e *Example) Handler(ctx context.Context, r interface{}) error {
	return nil
}

func TestDurable(t *testing.T) {

	//brkrSub := common.NewSubscribeOptions(
	//	common.Queue("queue.default"),
	//	common.DisableAutoAck(),
	//	rabbitmq.DurableQueue(),
	//)
	//
	//b := NewBroker(common.Addrs("amqp://rabbitmq:rabbitmq@127.0.0.1:5672"))
	//if err := b.Connect(); err != nil {
	//	t.Logf("cant conect to broker, skip: %v", err)
	//	t.Skip()
	//}

}
