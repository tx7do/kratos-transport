# Apache Pulsar

## 什么是Pulsar？

Apache Pulsar 是 Apache 软件基金会的顶级项目，是下一代云原生分布式消息流平台，集消息、存储、轻量化函数式计算为一体，采用计算与存储分离架构设计，支持多租户、持久化存储、多机房跨区域数据复制，具有强一致性、高吞吐、低延时及高可扩展性等流数据存储特性。

Pulsar 诞生于 2012 年，最初的目的是为在 Yahoo 内部，整合其他消息系统，构建统一逻辑、支撑大集群和跨区域的消息平台。当时的其他消息系统（包括 Kafka），都不能满足 Yahoo 的需求，比如大集群多租户、稳定可靠的 IO 服务质量、百万级 Topic、跨地域复制等，因此 Pulsar 应运而生。

Pulsar 于 2016 年底开源，现在是 Apache 软件基金会的一个孵化器项目。Pulsar 在 Yahoo 的生产环境运行了三年多，助力 Yahoo 的主要应用，如 Yahoo Mail、Yahoo Finance、Yahoo Sports、Flickr、Gemini 广告平台和 Yahoo 分布式键值存储系统 Sherpa。

## Pulsar 的关键特性

* 是下一代云原生分布式消息流平台。
* Pulsar 的单个实例原生支持多个集群，可跨机房在集群间无缝地完成消息复制。
* 极低地发布延迟和端到端延迟。
* 可无缝扩展到超过一百万个 topic。
* 简单的客户端 API，支持 Java、Go、Python 和 C++。
* 主题的多种订阅模式（独占、共享和故障转移）。
* 通过 Apache BookKeeper 提供的持久化消息存储机制保证消息传递 。
* 由轻量级的 serverless 计算框架 Pulsar Functions 实现流原生的数据处理。
* 基于 Pulsar Functions 的 serverless connector 框架 Pulsar IO 使得数据更易移入、移出 Apache Pulsar。
* 分层式存储可在数据陈旧时，将数据从热存储卸载到冷/长期存储（如S3、GCS）中。

## Pulsar 基本概念

### Producer

消息的源头，也是消息的发布者，负责将消息发送到 topic。

### Consumer

消息的消费者，负责从 topic 订阅并消费消息。

Pulsar具有3个订阅模式，它们可以共存在同一个主题上：

- **独享（exclusive）订阅** —— 同时只能有一个消费者。
- **共享（shared）订阅** —— 可以由多个消费者订阅，每个消费者接收其中的一部分消息。
- **失效备援（failover）订阅** —— 允许多个消费者连接到同一个主题上，但只有一个消费者能够接收消息。只有在当前消费者发生失效时，其他消费者才开始接收消息。

### Topic

消息数据的载体，在 Pulsar 中 Topic 可以指定分为多个 partition，如果不设置默认只有一个 partition。

### Broker

Broker 是一个无状态组件，主要负责接收 Producer 发送过来的消息，并交付给 Consumer。

### BookKeeper

分布式的预写日志系统，为消息系统比 Pulsar 提供存储服务，为多个数据中心提供跨机器复制。

### Bookie

Bookie 是为消息提供持久化的 Apache BookKeeper 的服务端。

### Cluster

Apache Pulsar 实例集群，由一个或多个实例组成。

## Docker部署开发环境

部署单机模式服务：

```shell
docker pull apachepulsar/pulsar:latest

docker run -itd \
    -p 6650:6650 \
    -p 8080:8080 \
    --name pulsar-standalone \
    apachepulsar/pulsar:latest bin/pulsar standalone
```

部署管理Web UI：

```shell
docker pull apachepulsar/pulsar-manager:latest

docker run -itd \
    -p 9527:9527 \
    -p 7750:7750 \
    --name pulsar-manager \
    -e SPRING_CONFIGURATION_FILE=/pulsar-manager/pulsar-manager/application.properties \
    apachepulsar/pulsar-manager:latest
```

设置管理员账号密码：

```shell
CSRF_TOKEN=$(curl http://localhost:7750/pulsar-manager/csrf-token)
curl \
   -H 'X-XSRF-TOKEN: $CSRF_TOKEN' \
   -H 'Cookie: XSRF-TOKEN=$CSRF_TOKEN;' \
   -H "Content-Type: application/json" \
   -X PUT http://localhost:7750/pulsar-manager/users/superuser \
   -d '{"name": "admin", "password": "apachepulsar", "description": "test", 
        "email": "username@test.org"}'
```

管理后台：<http://localhost:9527>  
账号：`admin`  
密码：`apachepulsar`

进入后台之后，在后台创建一个环境。其中，服务地址为pulsar的地址：

* 环境名（Environment Name）： `default`
* 服务地址（Service URL）：<http://host.docker.internal:8080>

## 参考资料

- [什么是消息队列？](https://aws.amazon.com/cn/message-queue/)
- [秒懂消息队列MQ，万字总结带你全面了解消息队列MQ](https://developer.aliyun.com/article/953777)
- [什么是消息队列？](https://www.ibm.com/cn-zh/topics/message-queues)
- [聊聊 Pulsar： Pulsar 的核心概念与基础架构](https://segmentfault.com/a/1190000041367545)
- [Apache Pulsar官网](https://pulsar.apache.org/)
- [发布订阅消息系统 Apache Pulsar 简介](https://www.infoq.cn/article/2017/11/apache-pulsar-brief-introduction)
- [一文读懂 Apache Pulsar](https://segmentfault.com/a/1190000041096450)
- [最佳实践｜Apache Pulsar 在拉卡拉的技术实践](https://xie.infoq.cn/article/babca7b9a2930c3c6d2d13126)
