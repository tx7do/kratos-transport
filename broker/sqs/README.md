# SQS

Amazon Simple Queue Service (Amazon SQS) 是一项 Web 服务，让您能够访问存储待处理消息的消息队列。借助 Amazon SQS，您能够快速构建可在任何计算机上运行的消息队列应用程序。

Amazon SQS 可以提供可靠、安全并且高度可扩展的托管队列服务，用于存储在计算机之间传输的消息。借助 Amazon SQS，您可以在不同的分布式应用程序组件之间移动数据，同时既不会丢失消息，也不需要各个组件始终处于可用状态。您可以使用与 AWS Key Management Service (KMS) 集成的 Amazon SQS 服务器端加密 (SSE) 在应用程序之间交换敏感数据。

Amazon SQS 可帮助您构建组件相互解耦的分布式应用程序，而且这些应用程序可与 Amazon Elastic Compute Cloud (Amazon EC2) 及其他 AWS 基础设施 Web 服务紧密配合。

## 队列类型

Amazon SQS 针对不同应用程序要求提供了两种队列类型：

### 标准队列

Amazon SQS 提供标准队列作为默认队列类型。标准队列使您能够每秒处理近乎无限数量的事务。标准队列可确保每条消息至少被传送一次。但是，由于允许高吞吐量的高度分布式架构，偶尔会有一条消息的某个副本不按顺序传送。标准队列提供最大努力排序，可保证消息大致按其发送的顺序进行传送。

### FIFO 队列 – 新功能！

FIFO 队列是对标准队列的补充。这种队列类型最重要的功能是 FIFO (先进先出) 传送和一次性处理：让消息的发送顺序和接收顺序严格保持一致，且消息只传送一次并保留到用户将其处理和删除；重复的消息不会被引入队列。FIFO 队列还支持消息组，即允许在单个队列中传送多个有序流。FIFO 队列的上限是 300 个事务/秒 (TPS)，但具有标准队列的全部功能。

## Docker部署开发服务器

Alpine SQS是第三方提供的开源实现。Alpine SQS 提供 Amazon Simple Queue Service (AWS-SQS) 的容器化 Java 实现。它基于运行 Alpine Linux 和 Oracle Java 8 Server-JRE 的 ElasticMQ。它与AWS的API、CLI以及Amazon Java SDK兼容。这可以加快本地开发速度，而无需承担基础设施成本。

```shell
docker pull roribio16/alpine-sqs:latest

docker run -d \
      --name sqs \
      -p 9324:9324 \
      -p 9325:9325 \
      roribio16/alpine-sqs:latest
```

## 参考资料

- [Amazon SQS 产品详情](https://aws.amazon.com/cn/sqs/details/?nc1=h_ls)
- [用 Docker 跑 SQS 範例隨記](https://lihan.cc/2021/11/1131/)
