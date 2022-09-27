# pulsar

Apache Pulsar 是 Apache 软件基金会的顶级项目，是下一代云原生分布式消息流平台，集消息、存储、轻量化函数式计算为一体，采用计算与存储分离架构设计，支持多租户、持久化存储、多机房跨区域数据复制，具有强一致性、高吞吐、低延时及高可扩展性等流数据存储特性。

Pulsar 的关键特性如下：

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

## 核心概念

### Messages（消息）
### Producers（生产者）
### Consumers（消费者）
### Topics（主题）
### Subscriptions（订阅模式）
### Message retention and expiry（消息保留和过期）
### Message deduplication（消息去重）

## Docker部署开发环境

```shell
docker pull apachepulsar/pulsar:latest

docker run -itd \
    -p 6650:6650 \
    -p 8080:8080 \
    --name pulsar-standalone \
    apachepulsar/pulsar:latest bin/pulsar standalone
```

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

管理后台 <http://localhost:9527>  
登陆账号：admin，密码：apachepulsar

在后台创建一个环境，其中，服务地址为pulsar的地址：
* 环境名（Environment Name）： default
* 服务地址（Service URL）：http://host.docker.internal:8080

## 参考资料

* [聊聊 Pulsar： Pulsar 的核心概念与基础架构](https://segmentfault.com/a/1190000041367545)
