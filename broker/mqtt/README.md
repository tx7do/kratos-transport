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
