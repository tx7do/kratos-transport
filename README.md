# kratos-transport

把Kafka等异步消息队列以及Websocket实现为微服务框架[Kratos](https://go-kratos.dev/docs/) 的transport.Server.  
在使用的时候,可以调用`kratos.Server()`方法,将之注册成为一个Server.

我另外有一个项目[kratos-cqrs](https://github.com/tx7do/kratos-cqrs) ,它引用了本项目,使用Kafka来消费数据库的写操作.其实要切到其他的队列也是简单的,比如已经支持的RabbitMQ,Redis.所需要进行的代码更改并不多,具体可以看例子和测试.

## 支持的队列或者协议

- [Kafka](https://kafka.apache.org/)
- [RabbitMQ](https://www.rabbitmq.com/)
- [NATS](https://nats.io/)
- [Redis](https://redis.io/)
- [MQTT](https://mqtt.org/)
- [WebSocket](https://zh.wikipedia.org/zh-hant/WebSocket)
