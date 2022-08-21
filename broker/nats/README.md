# NATS

NATS是由CloudFoundry的架构师Derek开发的一个开源的、轻量级、高性能的，支持发布、订阅机制的分布式消息队列系统。它的核心基于EventMachine开发，代码量不多，可以下载下来慢慢研究。其核心原理就是基于消息发布订阅机制。每个台服务 器上的每个模块会根据自己的消息类别，向MessageBus发布多个消息主题；而同时也向自己需要交互的模块，按照需要的信息内容的消息主题订阅消息。 NATS原来是使用Ruby编写，可以实现每秒150k消息，后来使用Go语言重写，能够达到每秒8-11百万个消息，整个程序很小只有3M Docker image，它不支持持久化消息，如果你离线，你就不能获得消息。

NATS适合云基础设施的消息通信系统、IoT设备消息通信和微服务架构。Apcera团队负责维护NATS服务器（Golang语言开发）和客户端（包括Go、Python、Ruby、Node.js、Elixir、Java、Nginx、C和C#），开源社区也贡献了一些客户端库，包括Rust、PHP、Lua等语言的库。目前已经采用了NATS系统的公司有：爱立信、HTC、百度、西门子、VMware。

## NATS的设计目标

NATS的设计原则是：高性能、可伸缩能力、易于使用，基于这些原则，NATS的设计目标包括：

1. 高性能（fast） 
2. 一直可用（dial tone） 
3. 极度轻量级（small footprint） 
4. 最多交付一次（fire and forget，消息发送后不管） 
5. 支持多种消息通信模型和用例场景（flexible）

## NATS应用场景

NATS理想的使用场景有：

1. 寻址、发现 
2. 命令和控制（控制面板） 
3. 负载均衡 
4. 多路可伸缩能力 
5. 定位透明 
6. 容错

NATS设计哲学认为，高质量的QoS应该在客户端构建，故只建立了请求-应答，不提供：

1. 持久化 
2. 事务处理 
3. 增强的交付模式 
4. 企业级队列

## 消息模式

支持3种消息模式：

* Publish/Subscribe
* Request/Reply
* Queueing

### Publish/Subscribe

Publish/Subscribe是一对多的消息模型。Publisher往一个主题上发送消息，任何订阅了此主题的Subscriber都可以接收到该主题的消息。

服务质量指标：

* 至多发一次

NATS系统是一种“发送后不管”的消息通信系统。往某主题上发送时，如果没有subscriber，或者所有subscriber不在线，则该消息不会给处理。如果需要更高的QoS，可以使用NATS Streaming，或者在客户端中增加可靠性。

* 至少发一次(NATS Streaming)

提供更高的的QoS，但是会付出降低吞吐率和增加延迟的代价。

### Request/Reply

publisher往主题中发布一个带预期响应的消息，subscriber执行请求调用，并返回最先的响应。 支持两种请求-响应消息通信模式：

* 点对点：最快、最先的响应。
* 一对多：可以限制Requestor收到的应答数量。

### Queueing

subscriber注册的时候，需指定一个队列名。指定相同队列名的subscriber，形成一个队列组。当主题收到消息后，订阅了此主题的队列组，会自动选择一个成员来接收消息。尽管队列组有多个subscriber，但每条消息只能被组中的一个subscriber接收。

## NATS Protocol

NATS连接协议是一个简单的、基于文本的发布/订阅风格的协议。与传统的二进制消息格式的消息通信系统不同，基于文本的NATS协议，使得客户端实现很简单，可以方便地选择多种编程语言或脚本语言来实现。

### 协议约定

#### 主题

大小写敏感，必须是不能包含空格的非空字符串，可以包含标志分隔符”.”。

#### 通配符

订阅主题中可以使用通配符，但是通配符必须被标识分隔。支持两种通配符：

星号*：匹配任意层级中的任意标记，如A.*.
大于号>：匹配所有当前层级之后的标记，如A.>

#### 新行

CR+LF（即\r\n，0X0D0A）作为协议消息的终止。新行还用于标记PUB或MSG协议中消息的实际有效负载的开始。

### 协议操作

操作名是大小写不敏感的。详细的操作，参考[NATS Protocol](http://nats.io/documentation/internals/nats-protocol/)

Client操作之后，Server都会给出相应的信息。

* `+OK`：Server响应正确。
* `-Err`：协议错误，将导致Client断开连接。


## Docker部署开发环境

```shell
docker pull bitnami/nats:latest
docker pull bitnami/nats-exporter:latest

docker run -itd \
    --name nats-server \
    --p 4222:4222 \
    --p 6222:6222 \
    --p 8000:8222 \
    -e NATS_HTTP_PORT_NUMBER=8222 \
    bitnami/nats:latest
```

管理后台: <https://127.0.0.1:8000>
