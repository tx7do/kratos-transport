# kratos-transport

把消息队列、任务队列，以及Websocket、HTTP3等网络协议实现为微服务框架 [Kratos](https://go-kratos.dev/docs/) 的
`transport.Server`。

在使用的时候,可以调用`kratos.Server()`方法，将之注册成为一个`Server`。

各种缝合，请叫我：缝合怪。

## 支持的服务（Server）

### 消息队列

- [RabbitMQ](./transport/rabbitmq/README.md)
- [Kafka](./transport/kafka/README.md)
- [RocketMQ](./transport/rocketmq/README.md)
- [ActiveMQ](./transport/activemq/README.md)
- [Pulsar](./transport/pulsar/README.md)
- [NATS](./transport/nats/README.md)
- [NSQ](./transport/nsq/README.md)
- [Redis](./transport/redis/README.md)
- [MQTT](./transport/mqtt/README.md)

### RPC

- [Thrift](./transport/thrift/README.md)
- [GraphQL](./transport/graphql/README.md)
- [FastHttp](./transport/fasthttp/README.md)
- [Gin](./transport/gin/README.md)
- [Go-Zero](./transport/gozero/README.md)
- [Hertz](./transport/hertz/README.md)
- [Iris](./transport/iris/README.md)

### 分布式任务队列

- [Asynq](./transport/hptimer/README.md)
- [Machinery](./transport/machinery/README.md)

### 网络协议

- [WebSocket](./transport/websocket/README.md)
- [HTTP3](./transport/http3/README.md)
- [WebTransport](./transport/webtransport/README.md)
- [SSE](./transport/sse/README.md)
- [SignalR](./transport/signalr/README.md)
- [Socket.IO](./transport/socketio/README.md)
- [MCP](./transport/mcp/README.md)
- [KCP](./transport/kcp/README.md)
- [WebRTC](./transport/webrtc/README.md)

### 定时器

- [hptimer](./transport/hptimer/README.md)
- [Cron](./transport/cron/README.md)

## 支持的消息代理（Broker）

- [Kafka](./broker/kafka/README.md)
- [MQTT](./broker/mqtt/README.md)
- [NATS](./broker/nats/README.md)
- [NSQ](./broker/nsq/README.md)
- [Pulsar](./broker/pulsar/README.md)
- [RabbitMQ](./broker/rabbitmq/README.md)
- [Redis](./broker/redis/README.md)
- [RocketMQ](./broker/rocketmq/README.md)
- [SQS](./broker/sqs/README.md)
- [STOMP](./broker/stomp/README.md)

## 应用示例

- [kratos-chatroom](https://github.com/tx7do/kratos-chatroom) 一个简单的Websocket聊天室的示例
- [kratos-cqrs](https://github.com/tx7do/kratos-cqrs) 一个CQRS架构模式的示例
- [kratos-realtimemap](https://github.com/tx7do/kratos-realtimemap) 一个物联网的公共交通实时显示地图的示例
- [kratos-uba](https://github.com/tx7do/kratos-uba) 一个UBA系统
- [go-kratos-admin](https://github.com/tx7do/go-kratos-admin) 一个Admin系统

以上示例在[Kratos官方示例代码库](https://github.com/go-kratos/examples)中也可以找到。
