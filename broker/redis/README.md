# Redis

## 什么是 Redis？

Redis 是一个基于内存的高性能 key-value 数据库。是完全开源免费的，用C语言编写的，遵守BSD协议。

Redis 特点：

- Redis 是基于内存操作的，吞吐量非常高，可以在 1s内完成十万次读写操作
- Redis 的读写模块是单线程，每个操作都具原子性
- Redis 支持数据的持久化，可以将内存中的数据保存在磁盘中，重启可以再次加载，但可能会有极短时间内数据丢失
- Redis 支持多种数据结构，String，list，set，zset，hash等

## Redis 发布订阅

Redis 发布订阅 (pub/sub) 是一种消息通信模式：发送者 (pub) 发送消息，订阅者 (sub) 接收消息。

Redis 的 SUBSCRIBE 命令可以让客户端订阅任意数量的频道， 每当有新信息发送到被订阅的频道时， 信息就会被发送给所有订阅指定频道的客户端。

## Redis发布订阅相关命令

利用了Redis的发布订阅命令

### PUBLISH channel message

用于将信息发送到指定的频道。

### SUBSCRIBE channel [channel …]

订阅给定的一个或多个频道的信息

### UNSUBSCRIBE [channel [channel …]]

指退订给定的频道

### PSUBSCRIBE pattern [pattern ...]

订阅一个或多个符合给定模式的频道

### PUBSUB subcommand [argument [argument ...]]

查看订阅与发布系统状态

### PUNSUBSCRIBE [pattern [pattern ...]]

退订所有给定模式的频道

## Docker部署开发环境

```shell
docker pull bitnami/redis:latest
docker pull bitnami/redis-exporter:latest

docker run -itd \
    --name redis-test \
    -p 6379:6379 \
    -e ALLOW_EMPTY_PASSWORD=yes \
    bitnami/redis:latest
```

## 管理工具

- [RedisInsight](https://redis.com/redis-enterprise/redis-insight/)
- [Another Redis Desktop Manager](https://github.com/qishibo/AnotherRedisDesktopManager/releases)

## 注意事项

- Redis无法对消息持久化存储，一旦消息被发送，而此时没有订阅者接收，那么消息就会丢失；
- 其他的一些消息队列（Kafka、Rabbitmq等）提供了消息传输保障，当客户端连接超时或事务回滚等情况发生时，消息会被重新发送给客户端，而Redis没有提供消息传输保障。
