# NSQ

NSQ是一个基于Go语言的分布式实时消息平台, 它具有分布式、去中心化的拓扑结构，支持无限水平扩展。无单点故障、故障容错、高可用性以及能够保证消息的可靠传递的特征。另外，NSQ非常容易配置和部署, 且支持众多的消息协议。支持多种客户端，协议简单，如果有兴趣可参照协议自已实现一个也可以。

## 基本概念

### Topic

一个topic就是程序发布消息的一个逻辑键，当程序第一次发布消息时就会创建topic。

### Channel

channel组与消费者相关，是消费者之间的负载均衡，channel在某种意义上来说是一个“队列”。每当一个发布者发送一条消息到一个topic，消息会被复制到所有消费者连接的channel上，消费者通过这个特殊的channel读取消息，实际上，在消费者第一次订阅时就会创建channel。

Channel会将消息进行排列，如果没有消费者读取消息，消息首先会在内存中排队，当量太大时就会被保存到磁盘中。

### Message

消息构成了我们数据流的中坚力量，消费者可以选择结束消息，表明它们正在被正常处理，或者重新将他们排队待到后面再进行处理。每个消息包含传递尝试的次数，当消息传递超过一定的阀值次数时，我们应该放弃这些消息，或者作为额外消息进行处理。

## 基本组件

### nsqlookupd

nsqlookupd服务器像consul或etcd那样工作，只是它被设计得没有协调和强一致性能力。每个nsqlookupd都作为nsqd节点注册信息的短暂数据存储区。消费者连接这些节点去检测需要从哪个nsqd节点上读取消息。

### nsqd

nsqd守护进程是NSQ的核心部分，它是一个单独的监听某个端口进来的消息的二进制程序。每个nsqd节点都独立运行，不共享任何状态。当一个节点启动时，它向一组nsqlookupd节点进行注册操作，并将保存在此节点上的topic和channel进行广播。

客户端可以发布消息到nsqd守护进程上，或者从nsqd守护进程上读取消息。通常，消息发布者会向一个单一的local nsqd发布消息，消费者从连接了的一组nsqd节点的topic上远程读取消息。如果你不关心动态添加节点功能，你可以直接运行standalone模式。

### nsqadmin

一套Web用户界面，可实时查看集群的统计数据和执行相应的管理任务

## 消息模式

NSQ的消息模式为推的方式，这种模式可以保证消息的及时性，当有消息时可以及时推送出去。但是要根椐客户端的消耗能力和节奏去控制，NSQ是通过更改RDY的值来实现的。当没有消息时为0, 服务端推送消息后，客户端比如调用 updateRDY（）这个方法改成3, 那么服务端推送时，就会根椐这个值做流控了。

NSQ还支持延时消息的发送，比如订单在30分钟未支付做无效处理等场景，延时使用的是heap包的优级先队列，实现了里面的一些方法。通过判断当前时间和延时时间做对比，然后从延时队列里面弹出消息再发送到channel中，后续流程和普通消息一样，我看网上有 人碰到过说延时消息会有并发问题，最后还用的Redis的ZSET实现的，所以不确定这个延时的靠不靠谱，要求不高的倒是可以试试。

## Docker部署开发环境

```shell
docker pull nsqio/nsq:latest

# nsqlookupd
docker run -d \
    --name nsqlookupd \
    -p 4160:4160 \
    -p 4161:4161 \
    nsqio/nsq:latest \
    /nsqlookupd

# nsqd
docker run -itd \
    --name nsqd \
    -p 4150:4150 \
    -p 4151:4151 \
    --link nsqlookupd \
    nsqio/nsq:latest \
    /nsqd --lookupd-tcp-address=nsqlookupd:4160 --broadcast-address=host.docker.internal

#nsqadmin
docker run -itd \
    --name nsqadmin \
    -p 4171:4171 \
    --link nsqlookupd \
    nsqio/nsq:latest \
    /nsqadmin --lookupd-http-address=nsqlookupd:4161
```

控制台访问地址： <http://127.0.0.1:4171>
直接使用REST API查看节点信息： <http://127.0.0.1:4161/nodes>
