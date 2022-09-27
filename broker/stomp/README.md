# STOMP

STOMP即Simple (or Streaming) Text Orientated Messaging Protocol，简单(流)
文本定向消息协议，它提供了一个可互操作的连接格式，允许STOMP客户端与任意STOMP消息代理（Broker）进行交互。STOMP协议由于设计简单，易于开发客户端，因此在多种语言和多种平台上得到广泛地应用。

STOMP协议的前身是TTMP协议（一个简单的基于文本的协议），专为消息中间件设计。

STOMP是一个非常简单和容易实现的协议，其设计灵感源自于HTTP的简单性。尽管STOMP协议在服务器端的实现可能有一定的难度，但客户端的实现却很容易。例如，可以使用Telnet登录到任何的STOMP代理，并与STOMP代理进行交互。

STOMP协议与2012年10月22日发布了最新的STOMP 1.2规范。
要查看STOMP 1.2规范，见： https://stomp.github.io/stomp-specification-1.2.html

## STOMP协议分析

STOMP协议与HTTP协议很相似，它基于TCP协议，使用了以下命令：  
CONNECT  
SEND  
SUBSCRIBE  
UNSUBSCRIBE  
BEGIN 
COMMIT  
ABORT  
ACK
NACK  
DISCONNECT

STOMP的客户端和服务器之间的通信是通过“帧”（Frame）实现的，每个帧由多“行”（Line）组成。  
第一行包含了命令，然后紧跟键值对形式的Header内容。  
第二行必须是空行。  
第三行开始就是Body内容，末尾都以空字符结尾。  
STOMP的客户端和服务器之间的通信是通过MESSAGE帧、RECEIPT帧或ERROR帧实现的，它们的格式相似。

## 支持STOMP的服务器

| 项目名             | 兼容STOMP的版本  | 描述                                                                                        |
|-----------------|-------------|-------------------------------------------------------------------------------------------|
| Apache Apollo   | 1.0 1.1 1.2 | ActiveMQ的继承者 http://activemq.apache.org/apollo                                            |
| Apache ActiveMQ | 1.0 1.1     | 流行的开源消息服务器 http://activemq.apache.org/                                                    |
| HornetQ         | 1.0         | 来自JBoss的消息中间件 http://www.jboss.org/hornetq                                                |
| RabbitMQ        | 1.0 1.1 1.2 | 基于Erlang、支持多种协议的消息Broker，通过插件支持STOMP协议http://www.rabbitmq.com/plugins.html#rabbitmq-stomp |
| Stampy          | 1.2         | STOMP 1.2规范的一个Java实现 http://mrstampy.github.com/Stampy/                                   |
| StompServer     | 1.0         | 一个轻量级的纯Ruby实现的STOMP服务器 http://stompserver.rubyforge.org/                                  |

## Docker部署开发服务器

### ActiveMQ

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

### Apache Apollo

```shell
```
