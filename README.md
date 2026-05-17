# kratos-transport

[![Stars](https://img.shields.io/github/stars/tx7do/kratos-transport?style=flat-square)](https://github.com/tx7do/kratos-transport/stargazers)
[![Last Commit](https://img.shields.io/github/last-commit/tx7do/kratos-transport?style=flat-square)](https://github.com/tx7do/kratos-transport/commits/main)
[![License](https://img.shields.io/github/license/tx7do/kratos-transport?style=flat-square)](./LICENSE)

为 [Kratos](https://go-kratos.dev/docs/) 扩展消息队列、任务队列与多种网络协议的 `transport.Server` / broker 能力。

## 目录
- [项目简介](#项目简介)
- [支持能力一览](#支持能力一览)
- [仓库结构](#仓库结构)
- [安装](#安装)
- [快速开始](#快速开始)
- [示例项目](#示例项目)
- [何时使用](#何时使用)
- [License](#license)

## 项目简介
`kratos-transport` 把消息队列、任务队列，以及 WebSocket、HTTP3、MCP 等协议封装为 Kratos 可接入的 transport / broker 组件，方便在统一的微服务框架中扩展更多通信方式。

如果你已经在使用 Kratos，希望把异步消息、实时通信或额外协议栈接入到现有服务，这个仓库就是一个集中式能力集合。

## 支持能力一览
### 消息队列 Server
- [RabbitMQ](./transport/rabbitmq/README.md)
- [Kafka](./transport/kafka/README.md)
- [RocketMQ](./transport/rocketmq/README.md)
- [ActiveMQ](./transport/activemq/README.md)
- [Pulsar](./transport/pulsar/README.md)
- [NATS](./transport/nats/README.md)
- [NSQ](./transport/nsq/README.md)
- [Redis](./transport/redis/README.md)
- [MQTT](./transport/mqtt/README.md)

### RPC / Web 扩展
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

### Broker
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

## 仓库结构
```text
broker/      多种消息代理实现
transport/   各类 transport.Server 扩展
_example/    示例工程
testing/     测试相关文件
tracing/     链路追踪相关扩展
script/      辅助脚本
```

## 安装
根据你需要的模块按需引入，例如：

```bash
go get github.com/tx7do/kratos-transport/transport/websocket
go get github.com/tx7do/kratos-transport/broker/kafka
```

## 快速开始
1. 选择你要接入的 transport 或 broker 模块。
2. 阅读对应子目录下的 README，例如 `transport/websocket/README.md`。
3. 在你的 Kratos 服务中注册对应组件。
4. 结合 `_example/` 中的示例验证运行效果。

对于 Kratos 应用，通常的接入思路是把扩展实现注册为一个 `Server`，再与已有 HTTP/gRPC 服务一起启动。

## 示例项目
- [kratos-chatroom](https://github.com/tx7do/kratos-chatroom) - WebSocket 聊天室示例
- [kratos-cqrs](https://github.com/tx7do/kratos-cqrs) - CQRS 架构示例
- [kratos-realtimemap](https://github.com/tx7do/kratos-realtimemap) - 物联网实时地图示例
- [go-wind-uba](https://github.com/tx7do/go-wind-uba) - UBA 系统
- [go-wind-admin](https://github.com/tx7do/go-wind-admin) - 中后台管理系统脚手架

以上示例在 [Kratos 官方 examples](https://github.com/go-kratos/examples) 中也可以找到相关参考。

## 何时使用
适合以下场景：
- 需要把消息队列或 broker 纳入 Kratos 统一抽象
- 需要给 Kratos 服务增加实时通信或额外协议支持
- 希望复用现成实现而不是从零封装 transport 层

## License
See [LICENSE](./LICENSE).
