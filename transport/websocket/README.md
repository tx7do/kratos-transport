# Websocket

## 什么是WebSocket?

WebSocket 协议主要为了解决基于 HTTP/1.x 的 Web 应用无法实现服务端向客户端主动推送的问题, 为了兼容现有的设施, WebSocket 协议使用与 HTTP 协议相同的端口, 并使用 HTTP Upgrade 机制来进行 WebSocket 握手, 当握手完成之后, 通信双方便可以按照 WebSocket 协议的方式进行交互

WebSocket 使用 TCP 作为传输层协议, 与 HTTP 类似, WebSocket 也支持在 TCP 上层引入 TLS 层, 以建立加密数据传输通道, 即 WebSocket over TLS, WebSocket 的 URI 与 HTTP URI 的结构类似, 对于使用 80 端口的 WebSocket over TCP, 其 URI 的一般形式为 `ws://host:port/path/query` 对于使用 443 端口的 WebSocket over TLS, 其 URI 的一般形式为 `wss://host:port/path/query`

在 WebSocket 协议中, 帧 (frame) 是通信双方数据传输的基本单元, 与其它网络协议相同, frame 由 Header 和 Payload 两部分构成, frame 有多种类型, frame 的类型由其头部的 Opcode 字段 (将在下面讨论) 来指示, WebSocket 的 frame 可以分为两类, 一类是用于传输控制信息的 frame (如通知对方关闭 WebSocket 连接), 一类是用于传输应用数据的 frame, 使用 WebSocket 协议通信的双方都需要首先进行握手, 只有当握手成功之后才开始使用 frame 传输数据

## 参考资料

* [RFC 6455 - The WebSocket Protocol](https://tools.ietf.org/html/rfc6455)
* [wikipedia - WebSocket](https://en.wikipedia.org/wiki/WebSocket)
* [HTML5 WebSocket](https://www.runoob.com/html/html5-websocket.html)
* [MDN - WebSocket](https://developer.mozilla.org/zh-CN/docs/Web/API/WebSocket)
* [WebSocket 协议解析 [RFC 6455]](https://sunyunqiang.com/blog/websocket_protocol_rfc6455/)
* [WebSocket 教程](https://www.ruanyifeng.com/blog/2017/05/websocket.html)
