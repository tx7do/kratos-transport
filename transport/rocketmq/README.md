# RocketMQ

## 什么是RocketMQ？

RocketMQ是由阿里捐赠给Apache的一款低延迟、高并发、高可用、高可靠的分布式消息中间件。经历了淘宝双十一的洗礼。RocketMQ既可为分布式应用系统提供异步解耦和削峰填谷的能力，同时也具备互联网应用所需的海量消息堆积、高吞吐、可靠重试等特性。

RocketMQ 特点:

- 是一个队列模型的消息中间件，具有高性能、高可靠、高实时、分布式等特点
- Producer、Consumer、队列都可以分布式
- Producer 向一些队列轮流发送消息，队列集合称为 Topic，Consumer 如果做广播消费，则一个 Consumer 实例消费这个 Topic 对应的所有队列，如果做集群消费，则多个 Consumer 实例平均消费这个 Topic 对应的队列集合
- 能够保证严格的消息顺序
- 支持拉（pull）和推（push）两种消息模式
- 高效的订阅者水平扩展能力
- 实时的消息订阅机制
- 亿级消息堆积能力
- 支持多种消息协议，如 JMS、OpenMessaging 等
- 较少的依赖

## 应用场景

- **削峰填谷**：诸如秒杀、抢红包、企业开门红等大型活动时皆会带来较高的流量脉冲，或因没做相应的保护而导致系统超负荷甚至崩溃，或因限制太过导致请求大量失败而影响用户体验，消息队列RocketMQ可提供削峰填谷的服务来解决该问题。
- **异步解耦**：交易系统作为淘宝和天猫主站最核心的系统，每笔交易订单数据的产生会引起几百个下游业务系统的关注，包括物流、购物车、积分、流计算分析等等，整体业务系统庞大而且复杂，消息队列RocketMQ可实现异步通信和应用解耦，确保主站业务的连续性。
- **顺序收发**：细数日常中需要保证顺序的应用场景非常多，例如证券交易过程时间优先原则，交易系统中的订单创建、支付、退款等流程，航班中的旅客登机消息处理等等。与先进先出FIFO（First In First Out）原理类似，消息队列RocketMQ提供的顺序消息即保证消息FIFO。
- **分布式事务一致性**：交易系统、支付红包等场景需要确保数据的最终一致性，大量引入消息队列RocketMQ的分布式事务，既可以实现系统之间的解耦，又可以保证最终的数据一致性。
- **大数据分析**：数据在“流动”中产生价值，传统数据分析大多是基于批量计算模型，而无法做到实时的数据分析，利用阿里云消息队列RocketMQ与流式计算引擎相结合，可以很方便的实现业务数据的实时分析。
- **分布式缓存同步**：天猫双11大促，各个分会场琳琅满目的商品需要实时感知价格变化，大量并发访问数据库导致会场页面响应时间长，集中式缓存因带宽瓶颈，限制了商品变更的访问流量，通过消息队列RocketMQ构建分布式缓存，实时通知商品数据的变化。

## 核心概念

- **Topic**：消息主题，一级消息类型，生产者向其发送消息。
- **Message**：生产者向Topic发送并最终传送给消费者的数据消息的载体。
- **消息属性**：生产者可以为消息定义的属性，包含Message Key和Tag。
- **Message Key**：消息的业务标识，由消息生产者（Producer）设置，唯一标识某个业务逻辑。
- **Message ID**：消息的全局唯一标识，由消息队列RocketMQ系统自动生成，唯一标识某条消息。
- **Tag**：消息标签，二级消息类型，用来进一步区分某个Topic下的消息分类
- **Producer**：也称为消息发布者，负责生产并发送消息至Topic。
- **Consumer**：也称为消息订阅者，负责从Topic接收并消费消息。
- **分区**：即Topic Partition，物理上的概念。每个Topic包含一个或多个分区。
- **消费位点**：每个Topic会有多个分区，每个分区会统计当前消息的总条数，这个称为最大位点MaxOffset；分区的起始位置对应的位置叫做起始位点MinOffset。
- **Group**：一类生产者或消费者，这类生产者或消费者通常生产或消费同一类消息，且消息发布或订阅的逻辑一致。
- **Group ID**：Group的标识。
- **队列**：个Topic下会由一到多个队列来存储消息。
- **Exactly-Once投递语义**：Exactly-Once投递语义是指发送到消息系统的消息只能被Consumer处理且仅处理一次，即使Producer重试消息发送导致某消息重复投递，该消息在Consumer也只被消费一次。
- **集群消费**：一个Group ID所标识的所有Consumer平均分摊消费消息。例如某个Topic有9条消息，一个Group ID有3个Consumer实例，那么在集群消费模式下每个实例平均分摊，只消费其中的3条消息。
- **广播消费**：一个Group ID所标识的所有Consumer都会各自消费某条消息一次。例如某个Topic有9条消息，一个Group ID有3个Consumer实例，那么在广播消费模式下每个实例都会各自消费9条消息。
- **定时消息**：Producer将消息发送到消息队列RocketMQ服务端，但并不期望这条消息立马投递，而是推迟到在当前时间点之后的某一个时间投递到Consumer进行消费，该消息即定时消息。
- **延时消息**：Producer将消息发送到消息队列RocketMQ服务端，但并不期望这条消息立马投递，而是延迟一定时间后才投递到Consumer进行消费，该消息即延时消息。
- **事务消息**：RocketMQ提供类似X/Open XA的分布事务功能，通过消息队列RocketMQ的事务消息能达到分布式事务的最终一致。
- **顺序消息**：RocketMQ提供的一种按照顺序进行发布和消费的消息类型，分为全局顺序消息和分区顺序消息。
- **全局顺序消息**：对于指定的一个Topic，所有消息按照严格的先入先出（FIFO）的顺序进行发布和消费。
- **分区顺序消息**：对于指定的一个Topic，所有消息根据Sharding Key进行区块分区。同一个分区内的消息按照严格的FIFO顺序进行发布和消费。Sharding Key是顺序消息中用来区分不同分区的关键字段，和普通消息的Message Key是完全不同的概念。
- **消息堆积**：Producer已经将消息发送到消息队列RocketMQ的服务端，但由于Consumer消费能力有限，未能在短时间内将所有消息正确消费掉，此时在消息队列RocketMQ的服务端保存着未被消费的消息，该状态即消息堆积。
- **消息过滤**：Consumer可以根据消息标签（Tag）对消息进行过滤，确保Consumer最终只接收被过滤后的消息类型。消息过滤在消息队列RocketMQ的服务端完成。
- **消息轨迹**：在一条消息从Producer发出到Consumer消费处理过程中，由各个相关节点的时间、地点等数据汇聚而成的完整链路信息。通过消息轨迹，您能清晰定位消息从Producer发出，经由消息队列RocketMQ服务端，投递给Consumer的完整链路，方便定位排查问题。
- **重置消费位点**：以时间轴为坐标，在消息持久化存储的时间范围内（默认3天），重新设置Consumer对已订阅的Topic的消费进度，设置完成后Consumer将接收设定时间点之后由Producer发送到消息队列RocketMQ服务端的消息。
- **死信队列**：死信队列用于处理无法被正常消费的消息。当一条消息初次消费失败，消息队列RocketMQ会自动进行消息重试；达到最大重试次数后，若消费依然失败，则表明Consumer在正常情况下无法正确地消费该消息。此时，消息队列RocketMQ不会立刻将消息丢弃，而是将这条消息发送到该Consumer对应的特殊队列中。

消息队列RocketMQ将这种正常情况下无法被消费的消息称为死信消息（Dead-Letter Message），将存储死信消息的特殊队列称为死信队列（Dead-Letter Queue）。

## RocketMQ 架构

RocketMQ 架构共有四个集群：NameServer 集群、Broker 集群、Producer 集群、Consumer 集群

### NameServer 集群

提供轻量级的服务发现及路由，每个 NameServer 记录完整的路由信息，提供相应的读写服务，支持快速存储扩展。有些其它开源中间件使用 ZooKeeper 实现服务发现及路由功能，如 Apache Kafka。

NameServer是一个功能齐全的服务器，主要包含两个功能：
1) Broker 管理，接收来自 Broker 集群的注册请求，提供心跳机制检测 Broker 是否存活
2) 路由管理，每个 NameServer 持有全部有关 Broker 集群和客户端请求队列的路由信息

### Broker 集群

通过提供轻量级的 Topic 和Queue 机制处理消息存储。同时支持推（Push）和拉（Pull）两种模型，包含容错机制。提供强大的峰值填充和以原始时间顺序累积数千亿条消息的能力。此外还提供灾难恢复，丰富的指标统计数据和警报机制，这些都是传统的消息系统缺乏的。
Broker 有几个重要的子模块：
1) 远程处理模块，Broker 入口，处理来自客户端的请求
2) 客户端管理，管理客户端（包括消息生产者和消费者），维护消费者的主题订阅
3) 存储服务，提供在物理硬盘上存储和查询消息的简单 API
4) HA 服务，提供主从 Broker 间数据同步
5) 索引服务，通过指定键为消息建立索引并提供快速消息查询

### Producer 集群

消息生产者支持分布式部署，分布式生产者通过多种负载均衡模式向 Broker 集群发送消息。

### Consumer 集群

消息消费者也支持 Push 和 Pull 模型的分布式部署，还支持集群消费和消息广播。提供了实时的消息订阅机制，可以满足大多数消费者的需求。

## 消息收发模型

消息队列RocketMQ支持发布和订阅模型，消息生产者应用创建Topic并将消息发送到Topic。消费者应用创建对Topic的订阅以便从其接收消息。通信可以是一对多（扇出）、多对一（扇入）和多对多。具体通信如下图所示。

- 生产者集群：用来表示发送消息应用，一个生产者集群下包含多个生产者实例，可以是多台机器，也可以是一台机器的多个进程，或者一个进程的多个生产者对象。

- 消费者集群：用来表示消费消息应用，一个消费者集群下包含多个消费者实例，可以是多台机器，也可以是多个进程，或者是一个进程的多个消费者对象。

一个生产者集群可以发送多个Topic消息。发送分布式事务消息时，如果生产者中途意外宕机，消息队列RocketMQ服务端会主动回调生产者集群的任意一台机器来确认事务状态。

一个消费者集群下的多个消费者以均摊方式消费消息。如果设置的是广播方式，那么这个消费者集群下的每个实例都消费全量数据。

一个消费者集群对应一个Group ID，一个Group ID可以订阅多个Topic，如上图中的Group 2所示。Group和Topic的订阅关系可以通过直接在程序中设置即可。

## Docker部署开发环境

必须要至少启动一个NameServer，一个Broker。

```shell
docker pull apache/rocketmq:latest

# NameServer
docker run -d \
      --name rmqnamesrv \
      -e "JAVA_OPT_EXT=-Xms512M -Xmx512M -Xmn128m" \
      -p 9876:9876 \
      apache/rocketmq:latest \
      sh mqnamesrv

# Broker
docker run -d \
      --name rmqbroker \
      -p 10911:10911 \
      -p 10909:10909 \
      -p 10912:10912 \
      --link rmqnamesrv \
      -e "JAVA_OPT_EXT=-Xms512M -Xmx512M -Xmn128m" \
      -e "NAMESRV_ADDR=rmqnamesrv:9876" \
      apache/rocketmq:latest \
      sh mqbroker -c /home/rocketmq/rocketmq-4.9.2/conf/broker.conf
```

```shell
docker pull styletang/rocketmq-console-ng:latest

docker run -d \
    --name rmqconsole \
    -p 9800:8080 \
    --link rmqnamesrv \
    -e "JAVA_OPTS=-Drocketmq.namesrv.addr=rmqnamesrv:9876 -Dcom.rocketmq.sendMessageWithVIPChannel=false" \
    -t styletang/rocketmq-console-ng:latest
```

控制台访问地址： <http://localhost:9800/#/>

另外，NameServer下发的是Docker容器的内网IP地址，从宿主机的外网访问是访问不了的，需要进行配置：

```bash
vi /home/rocketmq/rocketmq-4.9.2/conf/broker.conf
```

添加如下配置，brokerIP1可以是ip也可以是dns，hostname：

```ini
brokerIP1 = host.docker.internal
```

## 参考资料

* [什么是消息队列RocketMQ版？](https://help.aliyun.com/document_detail/29532.html?userCode=qtldtin2)
* [RocketMQ 介绍及核心概念](https://www.jianshu.com/p/2ae8e81718d3)
* [RocketMQ 简介](https://segmentfault.com/a/1190000038844218)
