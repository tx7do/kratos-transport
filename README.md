# kratos-transport

把Kafka等异步消息队列以及Websocket实现为微服务框架[Kratos](https://go-kratos.dev/docs/) 的`transport.Server`。

在使用的时候,可以调用`kratos.Server()`方法，将之注册成为一个`Server`。

我另外有一个项目[kratos-cqrs](https://github.com/tx7do/kratos-cqrs) ，它引用了本项目，使用Kafka来消费数据库的写操作。

要切到其他的队列（协议）也是简单的，只需要切换包即可，所需的代码更改并不多，具体可以看例子和测试。

## 支持的协议

- [MQTT](https://mqtt.org/)
- [WebSocket](https://zh.wikipedia.org/zh-hant/WebSocket)
- [STOMP](https://stomp.github.io/)
- [AMQP](https://www.amqp.org/)

## 支持的队列

- [RabbitMQ](https://www.rabbitmq.com/)
- [Kafka](https://kafka.apache.org/)
- [RocketMQ](https://rocketmq.apache.org/)
- [ActiveMQ](http://activemq.apache.org)
- [Apollo](http://activemq.apache.org/apollo)
- [Pulsar](https://pulsar.apache.org/)
- [NATS](https://nats.io/)
- [NSQ](https://nsq.io/)
- [Redis](https://redis.io/)
