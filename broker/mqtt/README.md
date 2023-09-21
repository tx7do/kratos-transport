# MQTT

## 什么是MQTT？

MQTT 协议 是由`IBM`的`Andy Stanford-Clark博士`和`Arcom`（已更名为Eurotech）的`Arlen Nipper博士`于 1999 年发明，用于石油和天然气行业。工程师需要一种协议来实现最小带宽和最小电池损耗，以通过卫星监控石油管道。最初，该协议被称为消息队列遥测传输，得名于首先支持其初始阶段的 IBM 产品 MQ 系列。2010 年，IBM 发布了 MQTT 3.1 作为任何人都可以实施的免费开放协议，然后于 2013 年将其提交给结构化信息标准促进组织 (OASIS) 规范机构进行维护。2019 年，OASIS 发布了升级的 MQTT 版本 5。MQTT最初代表的意思是 **Message Queueing Telemetry Transport（消息队列遥测传输）**，现在 MQTT 不再是首字母缩写词，而是被认为是协议的正式名称。

由于MQTT协议的通讯数据很精简，非常适用于CPU资源及网络带宽有限的物联网设备，再加上已经有许多MQTT程序库被陆续开发出来，用于Arduino控制板（C/C++ ）、JavaScript(Node.js, Espruino控制板), Python……等等，还有开放源代码的MQTT服务器，使得开发物联网（Internet of Things，IoT）、机器对机器（Machine-to-Machine，M2M）的通讯变得非常简单。Facebook Messenger也是使用的MQTT协议。

## 什么是 MQTT over WSS？

MQTT over WebSockets (WSS) 是一种 MQTT 实施，用于将数据直接接收到 Web 浏览器中。MQTT 协议定义了一个 JavaScript 客户端来为浏览器提供 WSS 支持。在这种情况下，该协议照常工作，但它向 MQTT 消息添加了额外标头以支持 WSS 协议。您可以将其视为包装在 WSS 信封中的 MQTT 消息负载。

## MQTT 背后的原理是什么？

MQTT 协议基于发布/订阅模型工作。在传统的网络通信中，客户端和服务器直接相互通信。客户端向服务器请求资源或数据，然后，服务器将处理并发回响应。但是，MQTT 使用发布/订阅模式将消息发送者（发布者）与消息接收者（订阅者）解耦。相反，称为消息代理的第三个组件将处理发布者和订阅者之间的通信。代理的工作是筛选所有来自发布者的传入消息，并将它们正确地分发给订阅者。代理将发布者和订阅者解耦，如下所示：

### 空间解耦

发布者和订阅者不知道彼此的网络位置，也不交换 IP 地址或端口号等信息。

### 时间解耦

发布者和订阅者不会同时运行或具有网络连接。

### 同步解耦

发布者和订阅者都可以发送或接收消息，而不会互相干扰。例如，订阅者不必等待发布者发送消息。

## MQTT 有哪些组成部分？

MQTT 通过如下定义客户端和代理来实施发布/订阅模型。

### MQTT 客户端

MQTT 客户端是从服务器到运行 MQTT 库的微控制器的任何设备。如果客户端正在发送消息，它将充当发布者；如果它正在接收消息，它将充当接收者。基本上，任何通过网络使用 MQTT 进行通信的设备都可以称为 MQTT 客户端设备。

### MQTT 代理

MQTT 代理是协调不同客户端之间消息的后端系统。代理的职责包括接收和筛选消息、识别订阅每条消息的客户端，以及向他们发送消息。它还负责其他任务，例如：

- 授权 MQTT 客户端以及对其进行身份验证
- 将消息传递给其他系统以进行进一步分析
- 处理错过的消息和客户端会话

###  MQTT 连接

客户端和代理开始使用 MQTT 连接进行通信。客户端通过向 MQTT 代理发送 CONNECT 消息来启动连接。代理通过响应 CONNACK 消息来确认已建立连接。MQTT 客户端和代理都需要 TCP/IP 堆栈进行通信。客户端从不相互联系，它们只与代理联系。

## MQTT 的工作原理？

下面概述了 MQTT 的工作原理。

1. MQTT 客户端与 MQTT 代理建立连接。
2. 连接后，客户端可以发布消息、订阅特定消息或同时执行这两项操作。
3. MQTT 代理收到一条消息后，会将其转发给对此感兴趣的订阅者。

让我们进行详细的分解，以进一步了解详情。

### MQTT 主题

“主题(Topic)”一词是指 MQTT 代理用于为 MQTT 客户端筛选消息的关键字。主题是分层组织的，类似于文件或文件夹目录。例如，考虑在多层房屋中运行的智能家居系统，每层都有不同的智能设备。在这种情况下，MQTT 代理可以将主题组织为：

> ourhome/groundfloor/livingroom/light  
> ourhome/firstfloor/kitchen/temperature

### MQTT publish

MQTT 客户端以字节格式发布包含主题和数据的消息。客户端确定数据格式，例如文本数据、二进制数据、XML 或 JSON 文件。例如，智能家居系统中的灯可能会针对主题 livingroom/light 发布消息 on。

### MQTT subscribe

MQTT 客户端向 MQTT 代理发送 SUBSCRIBE 消息，以接收有关感兴趣主题的消息。此消息包含唯一标识符和订阅列表。例如，您手机上的智能家居应用程序想要显示您家中有多少灯亮着。它将订阅主题 light 并增加所有 on 消息的计数器。

## MQTT 设计规范

由于物联网的环境是非常特别的，所以MQTT遵循以下设计原则：

1. 精简，不添加可有可无的功能； 
2. 发布/订阅（Pub/Sub）模式，方便消息在传感器之间传递； 
3. 允许用户动态创建主题，零运维成本； 
4. 把传输量降到最低以提高传输效率； 
5. 把低带宽、高延迟、不稳定的网络等因素考虑在内； 
6. 支持连续的会话控制； 
7. 理解客户端计算能力可能很低； 
8. 提供服务质量管理； 
9. 假设数据不可知，不强求传输数据的类型与格式，保持灵活性。

## MQTT 主要特性

MQTT协议工作在低带宽、不可靠的网络的远程传感器和控制设备通讯而设计的协议，它具有以下主要的几项特性：

### （1）使用发布/订阅消息模式，提供一对多的消息发布，解除应用程序耦合。

这一点很类似于XMPP，但是MQTT的信息冗余远小于XMPP，,因为XMPP使用XML格式文本来传递数据。

### （2）对负载内容屏蔽的消息传输。

### （3）使用TCP/IP提供网络连接。

主流的MQTT是基于TCP连接进行数据推送的，但是同样有基于UDP的版本，叫做MQTT-SN。这两种版本由于基于不同的连接方式，优缺点自然也就各有不同了。

### （4）有三种消息发布服务质量：

- "至多一次"，消息发布完全依赖底层TCP/IP网络。会发生消息丢失或重复。这一级别可用于如下情况，环境传感器数据，丢失一次读记录无所谓，因为不久后还会有第二次发送。这一种方式主要普通APP的推送，倘若你的智能设备在消息推送时未联网，推送过去没收到，再次联网也就收不到了。
- "至少一次"，确保消息到达，但消息重复可能会发生。
- "只有一次"，确保消息到达一次。在一些要求比较严格的计费系统中，可以使用此级别。在计费系统中，消息重复或丢失会导致不正确的结果。这种最高质量的消息发布服务还可以用于即时通讯类的APP的推送，确保用户收到且只会收到一次。

### （5）小型传输，开销很小（固定长度的头部是2字节），协议交换最小化，以降低网络流量。

这就是为什么在介绍里说它非常适合"在物联网领域，传感器与服务器的通信，信息的收集"，要知道嵌入式设备的运算能力和带宽都相对薄弱，使用这种协议来传递消息再适合不过了。

### （6）使用Last Will和Testament特性通知有关各方客户端异常中断的机制。

- Last Will：即遗言机制，用于通知同一主题下的其他设备发送遗言的设备已经断开了连接。
- Testament：遗嘱机制，功能类似于Last Will。

## MQTT协议中的方法

MQTT协议中定义了一些方法（也被称为动作），来于表示对确定资源所进行操作。这个资源可以代表预先存在的数据或动态生成数据，这取决于服务器的实现。通常来说，资源指服务器上的文件或输出。主要方法有：

1. Connect。等待与服务器建立连接。 
2. Disconnect。等待MQTT客户端完成所做的工作，并与服务器断开TCP/IP会话。 
3. Subscribe。等待完成订阅。
4. UnSubscribe。等待服务器取消客户端的一个或多个topics订阅。
5. Publish。MQTT客户端发送消息请求，发送完成后返回应用程序线程。

## 基本概念

### Message ID

消息的全局唯一标识，由微消息队列MQTT版系统自动生成，唯一标识某条消息。Message ID可用于回溯消息轨迹，排查问题。

### MQTT服务器

MQTT服务器可以称为"消息代理"（Broker），可以是一个应用程序或一台设备。它是位于消息发布者和订阅者之间，它可以：

1. 接受来自客户的网络连接； 
2. 接受客户发布的应用信息； 
3. 处理来自客户端的订阅和退订请求； 
4. 向订阅的客户转发应用程序消息。

### MQTT客户端

一个使用MQTT协议的应用程序或者设备，它总是建立到服务器的网络连接。客户端可以：

1. 发布其他客户端可能会订阅的信息；
2. 订阅其它客户端发布的消息；
3. 退订或删除应用程序的消息；
4. 断开与服务器连接。

### 消息服务质量(QoS)

QoS（Quality of Service）指消息传输的服务质量。分别可在消息发送端和消息消费端设置。

- 发送端的QoS设置：影响发送端发送消息到微消息队列MQTT版的传输质量。
- 消费端的QoS设置：影响微消息队列MQTT版服务端投递消息到消费端的传输质量。

QoS包括以下级别：

- QoS0：代表最多分发一次。
- QoS1：代表至少达到一次。
- QoS2：代表仅分发一次。

### 订阅（Subscription）

订阅包含`主题筛选器（Topic Filter）`和最大`消息服务质量（QoS）`。订阅会与一个`会话（Session）`关联。一个会话可以包含多个订阅。每一个会话中的每个订阅都有一个不同的主题筛选器。

### 会话（Session）

每个客户端与服务器建立连接后就是一个会话，客户端和服务器之间有状态交互。会话存在于一个网络之间，也可能在客户端和服务器之间跨越多个连续的网络连接。

### 主题名（Topic Name）

连接到一个应用程序消息的标签，该标签与服务器的订阅相匹配。服务器会将消息发送给订阅所匹配标签的每个客户端。

### 主题筛选器（Topic Filter）

一个对主题名通配符筛选器，在订阅表达式中使用，表示订阅所匹配到的多个主题。

### 负载（Payload）

消息订阅者所具体接收的内容。

### cleanSession

cleanSession标志是MQTT协议中对一个消费者客户端建立TCP连接后是否关心之前状态的定义，与消息发送端的设置无关。具体语义如下：

- cleanSession=true：消费者客户端再次上线时，将不再关心之前所有的订阅关系以及离线消息。
- cleanSession=false：消费者客户端再次上线时，还需要处理之前的离线消息，而之前的订阅关系也会持续生效。

消费端QoS和cleanSession的不同组合产生的结果如表 1所示。

表 1. QoS和cleanSession的组合关系

| QoS级别	 | cleanSession=true	 | cleanSession=false |
|--------|--------------------|--------------------|
| QoS0   | 无离线消息，在线消息只尝试推一次。  | 无离线消息，在线消息只尝试推一次。  |
| QoS1   | 无离线消息，在线消息保证可达。    | 有离线消息，所有消息保证可达。    |
| QoS2   | 无离线消息，在线消息保证只推一次。  | 暂不支持。              |

## Docker部署开发环境

### RabbitMQ

```shell
docker pull bitnami/rabbitmq:latest

docker run -itd \
    --hostname localhost \
    --name rabbitmq-test \
    -p 15672:15672 \
    -p 5672:5672 \
    -p 1883:1883 \
    -p 15675:15675 \
    -e RABBITMQ_PLUGINS=rabbitmq_top,rabbitmq_mqtt,rabbitmq_web_mqtt,rabbitmq_prometheus,rabbitmq_stomp,rabbitmq_auth_backend_http \
    bitnami/rabbitmq:latest

# 查看插件列表
rabbitmq-plugins list
# rabbitmq_peer_discovery_consul 
rabbitmq-plugins --offline enable rabbitmq_peer_discovery_consul
# rabbitmq_mqtt 提供与后端服务交互使用，端口1883
rabbitmq-plugins enable rabbitmq_mqtt
# rabbitmq_web_mqtt 提供与前端交互使用，端口15675
rabbitmq-plugins enable rabbitmq_web_mqtt
# 
rabbitmq-plugins enable rabbitmq_auth_backend_http
```

管理后台: <http://localhost:15672>  
默认账号: user  
默认密码: bitnami

### mosquitto

```shell
docker pull eclipse-mosquitto:latest

# 1883 tcp
# 9001 websockets
docker run -itd \
    --name mosquitto-test \
    -p 1883:1883 \
    -p 9001:9001 \
    eclipse-mosquitto:latest
```

### EMX

```shell
docker pull emqx/emqx:latest

docker run -itd \
    --name emqx-test \
    --add-host=host.docker.internal:host-gateway \
    -p 18083:18083 \
    -p 1883:1883 \
    emqx/emqx:latest
```

管理后台: <http://localhost:18083>  
默认账号: admin  
默认密码: public

### HiveMQ

```bash
docker pull hivemq/hivemq4:latest

docker run -itd \
    --name hivemq-test \
    --ulimit nofile=500000:500000 \
    -p 8080:8080 \
    -p 8000:8000 \
    -p 1883:1883 \
    hivemq/hivemq4:latest
```

## 热门在线公共 MQTT 服务器

| 名称	        | Broker 地址	               | TCP  | TLS         | WebSocket |
|------------|--------------------------|------|-------------|-----------|
| EMQ X	     | broker.emqx.io	          | 1883 | 8883        | 8083,8084 |
| EMQ X（国内）	 | broker-cn.emqx.io	       | 1883 | 8883        | 8083,8084 |
| Eclipse    | mqtt.eclipseprojects.io	 | 1883 | 8883        | 80, 443   |
| Mosquitto  | test.mosquitto.org	      | 1883 | 8883, 8884	 | 80        |
| HiveMQ     | broker.hivemq.com	       | 1883 | N/A	        | 8000      |

## 参考资料

- [MQTT教學（一）：認識MQTT](https://swf.com.tw/?p=1002)
- [MQTT 入门介绍](https://www.runoob.com/w3cnote/mqtt-intro.html)
- [什么是 MQTT？](https://aws.amazon.com/cn/what-is/mqtt/?nc1=h_ls)
