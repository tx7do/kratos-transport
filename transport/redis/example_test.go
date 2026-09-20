package redis_test

import (
	"context"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/broker"
	api "github.com/tx7do/kratos-transport/testing/api/manual"
	"github.com/tx7do/kratos-transport/transport/redis"
)

func handleHygrothermograph(_ context.Context, topic string, headers broker.Headers, msg *api.Hygrothermograph) error {
	log.Infof("Topic %s, Headers: %+v, Payload: %+v\n", topic, headers, msg)
	return nil
}

// ExampleNewServer 演示把 Redis 订阅挂到 kratos 应用上。
// 没有 Output 注释，go test 只编译不执行；实际运行需要本地 Redis (127.0.0.1:6379)。
func ExampleNewServer() {
	redisSrv := redis.NewServer(
		redis.WithAddress("127.0.0.1:6379"),
		redis.WithCodec("json"),
	)

	_ = redis.RegisterSubscriber(redisSrv, "test_topic", handleHygrothermograph)

	app := kratos.New(
		kratos.Name("redis"),
		kratos.Server(
			redisSrv,
		),
	)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}
