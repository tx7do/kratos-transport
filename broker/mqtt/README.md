# MQTT

MQTT(Message Queuing Telemetry Transport，消息队列遥测传输) 协议是IBM 针对物联网应用定制的即时通讯协议。
该协议使用TCP/IP 提供网络连接，能够对负载内容实现消息屏蔽传输，开销小，可以有效降低网络流量。

## 热门在线公共 MQTT 服务器

| 名称	        | Broker 地址	               | TCP  | TLS         | WebSocket |
|------------|--------------------------|------|-------------|-----------|
| EMQ X	     | broker.emqx.io	          | 1883 | 8883        | 8083,8084 |
| EMQ X（国内）	 | broker-cn.emqx.io	       | 1883 | 8883        | 8083,8084 |
| Eclipse    | mqtt.eclipseprojects.io	 | 1883 | 8883        | 80, 443   |
| Mosquitto  | test.mosquitto.org	      | 1883 | 8883, 8884	 | 80        |
| HiveMQ     | broker.hivemq.com	       | 1883 | N/A	        | 8000      |

## 基本概念

### Message ID

消息的全局唯一标识，由微消息队列MQTT版系统自动生成，唯一标识某条消息。Message ID可用于回溯消息轨迹，排查问题。

### MQTT服务器

微消息队列MQTT版提供的MQTT协议交互的服务端节点，用于完成与MQTT客户端和消息队列各自的消息收发。

###  MQTT客户端

用于和MQTT服务器交互的移动端节点，全称为微消息队列MQTT版客户端。

### QoS

QoS（Quality of Service）指消息传输的服务质量。分别可在消息发送端和消息消费端设置。

- 发送端的QoS设置：影响发送端发送消息到微消息队列MQTT版的传输质量。
- 消费端的QoS设置：影响微消息队列MQTT版服务端投递消息到消费端的传输质量。

QoS包括以下级别：

- QoS0：代表最多分发一次。
- QoS1：代表至少达到一次。
- QoS2：代表仅分发一次。

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
