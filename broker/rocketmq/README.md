# RocketMQ

## 消费方式

RocketMQ 提供了两类消费方式：PUSH 和 PULL。 PushConsumer：在大多数场景下使用，实时性更好。名字虽然是 Push 开头，实际在实现时，使用 Pull 方式实现。通过 Pull 不断不断不断轮询 Broker
获取消息。当不存在新消息时，Broker会挂起请求，直到有新消息产生，取消挂起，返回新消息。这样，基本和 Broker 主动 Push 做到接近的实时性（当然，还是有相应的实时性损失）。原理类似 长轮询( Long-Polling )。

## 禁用日志

日志禁用比较麻烦，它的日志等级配置只能通过环境变量设置，而不能通过配置文件设置。

```bash
export ROCKETMQ_GO_LOG_LEVEL=error
```

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
