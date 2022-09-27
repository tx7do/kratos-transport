# ActiveMQ

##  什么是ActiveMQ？

ActiveMQ 是 Apache 出品，最流行的，能力强劲的开源消息总线。ActiveMQ 是一个 完全支持 JMS1.1 和 J2EE 1.4 规范的 JMS Provider 实现，尽管 JMS 规范出台已经是很久 的事情了，但是 JMS 在当今的 J2EE 应用中间仍然扮演着特殊的地位。

ActiveMQ的主要特点如下：

* 支持Java消息服务(JMS) 1.1 版本
* 支持的编程语言包括：C、C++、C#、Delphi、Erlang、Adobe Flash、Haskell、Java、JavaScript、Perl、PHP、Pike、Python和Ruby；
* 协议支持包括：OpenWire、REST、STOMP、WS-Notification、MQTT、XMPP以及AMQP
* 集群 (Clustering)

## Docker部署开发环境

```shell
docker pull rmohr/activemq:latest

docker run -d \
      --name activemq-test \
      -p 61616:61616 \
      -p 8161:8161 \
      -p 5672:5672 \
      -p 61613:61613 \
      -p 1883:1883 \
      -p 61614:61614 \
      rmohr/activemq:latest
```

| 端口号   | 协议    |
|-------|-------|
| 61616 | JMS   |
| 8161  | UI    |
| 5672  | AMQP  |
| 61613 | STOMP |
| 1883  | MQTT  |
| 61614 | WS    |

* 管理后台：<http://localhost:8161/admin/>
* 默认账号名密码：admin/admin
