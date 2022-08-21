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
