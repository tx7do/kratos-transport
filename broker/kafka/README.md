# Kafka

Kafka是一个分布式流处理系统，流处理系统使它可以像消息队列一样publish或者subscribe消息，分布式提供了容错性，并发处理消息的机制。

## Kafka的基本概念

kafka运行在集群上，集群包含一个或多个服务器。kafka把消息存在topic中，每一条消息包含键值（key），值（value）和时间戳（timestamp）。

kafka有以下一些基本概念：

* **Producer** - 消息生产者，就是向kafka broker发消息的客户端。

* **Consumer** - 消息消费者，是消息的使用方，负责消费Kafka服务器上的消息。

* **Topic** - 主题，由用户定义并配置在Kafka服务器，用于建立Producer和Consumer之间的订阅关系。生产者发送消息到指定的Topic下，消息者从这个Topic下消费消息。

* **Partition** - 消息分区，一个topic可以分为多个 partition，每个partition是一个有序的队列。partition中的每条消息都会被分配一个有序的id（offset）。

* **Broker** - 一台kafka服务器就是一个broker。一个集群由多个broker组成。一个broker可以容纳多个topic。

* **Consumer Group** - 消费者分组，用于归组同类消费者。每个consumer属于一个特定的consumer group，多个消费者可以共同消息一个Topic下的消息，每个消费者消费其中的部分消息，这些消费者就组成了一个分组，拥有同一个分组名称，通常也被称为消费者集群。

* **Offset** - 消息在partition中的偏移量。每一条消息在partition都有唯一的偏移量，消息者可以指定偏移量来指定要消费的消息。

## Docker部署开发环境

```shell
docker pull bitnami/kafka:latest
docker pull bitnami/zookeeper:latest
docker pull bitnami/kafka-exporter:latest

docker run -itd \
    --name zookeeper-test \
    -p 2181:2181 \
    -e ALLOW_ANONYMOUS_LOGIN=yes \
    bitnami/zookeeper:latest

docker run -itd \
    --name kafka-standalone \
    --link zookeeper-test \
    -p 9092:9092 \
    -v /home/data/kafka:/bitnami/kafka \
    -e KAFKA_BROKER_ID=1 \
    -e KAFKA_LISTENERS=PLAINTEXT://:9092 \
    -e KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://127.0.0.1:9092 \
    -e KAFKA_ZOOKEEPER_CONNECT=zookeeper-test:2181 \
    -e ALLOW_PLAINTEXT_LISTENER=yes \
    --user root \
    bitnami/kafka:latest
```

## 管理工具

- [Offset Explorer](https://www.kafkatool.com/download.html)

## 参考资料

* [使用kafka-go导致的消费延时问题](https://loesspie.com/2020/12/28/kafka-golang-segmentio-kafka-go-slow-cousume/)
* [kafka-go 读取kafka消息丢失数据的问题定位和解决](https://cloud.tencent.com/developer/article/1809467)
* [Go社区主流Kafka客户端简要对比](https://tonybai.com/2022/03/28/the-comparison-of-the-go-community-leading-kakfa-clients/)
* [kafka go Writer 写入消息过慢的原因分析](http://timd.cn/kafka-go-writer/)
