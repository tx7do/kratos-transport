# WebTransport

## WebTransport 是什么？

WebTransport 是一个协议框架，该协议使客户端与远程服务器在安全模型下通信，并且采用安全多路复用传输技术。最新版的WebTransport草案中，该协议是基于HTTP3的，即WebTransport可天然复用QUIC和HTTP3的特性。

它有以下几种机制交换数据：

- Client 创建一个由特殊的不限制长度的 HTTP3 frame 组成的双向流（bidirectional stream），然后转移该流的所有权给 WebTransport；
- Server 创建一个 bidirectional stream，因为 HTTP3 没有为 Server 端创建 bidirectional stream 定义任何语义；
- Client 和 Server 都可以创建单向流类型（unidirectional stream）；
- 通过 QUIC DATAGRAM 帧发送数据包 datagram。


## QUIC 特性

- **用户态构建：** QUIC是在用户层构建的，所以不需要每次协议升级时进行内核修改。同时由于用户态可构建，QUIC的开源项目十分之多（具体可参见QUIC 开源实现列表 - 知乎 | https://zhuanlan.zhihu.com/p/270628018）。

- **流复用和流控：** QUIC引入了连接上的多路流复用的概念。QUIC通过设计实现了单独的、针对每个流的流控，解决了整个连接的队头阻塞问题。

- **灵活的拥塞控制机制：** TCP的拥塞控制机制是刚性的。该协议每次检测到拥塞时，都会将拥塞窗口大小减少一半。相比之下，QUIC的拥塞控制设计得更加灵活，可以更有效地利用可用的网络带宽，从而获得更好的吞吐量。

- **链接迁移：** 当客户端IP或端口发生变化时（这在移动端比较常见），TCP连接基于两端的ip:port四元组标示，因此会重新握手，而UDP不面向连接，不用握手。其上层的QUIC链路由于使用了64位Connection id作为唯一标识，四元组变化不影响链路的标示，也不用重新握手。

- **更好的错误处理能力：** QUIC使用增强的丢失恢复机制和转发纠错功能，以更好地处理错误数据包。该功能对于那些只能通过缓慢的无线网络访问互联网的用户来说是一个福音，因为这些网络用户在传输过程中经常出现高错误率。

- **更快的握手：** QUIC使用相同的TLS模块进行安全连接。然而，与TCP不同的是，QUIC的握手机制经过优化，避免了每次两个已知的对等者之间建立通信时的冗余协议交换，甚至可以做到0RTT握手。

- **更灵活的传输方式：** QUIC具有重传功能，可以提供可靠传输（stream），又因为基于UDP，也可提供非可靠传输（datagram）

## WebTransport 特性

Webtransport 基于 QUIC 协议，其底层是 UDP。虽然是 UDP 是不可靠的传输协议，但是 QUIC 在 UDP 的基础上融合了 TCP、TLS、HTTP/2 等协议的特性，使得 QUIC 成为一种低时延、安全可靠的传输协议。可以简单理解 QUIC 把 TCP+TLS 的功能基于 UDP 重新实现了一遍。

WebTransport 提供了如下功能特性：

- 传输可靠数据流 （类似 TCP）
- 传输不可靠数据流（类似 UDP）
- 数据加密和拥塞控制（congestion control）
- 基于 Origin 的安全模型（校验请求方是否在白名单内，类似于 CORS 的Access-Control-Allow-Origin）
- 支持多路复用（类似于 HTTP2 的 Stream）

## WebTransport 与 WebSocket 的区别

WebTransport 与 WebSocket最直接的区别就是一个是基于UDP，一个是基于TCP。WebSocket基于TCP，从而也自带TCP协议本身的缺陷，比如队头阻塞。

WebSocket 是基于消息的协议，所以你不需要考虑数据分包，封装的问题。WebTransport 是基于流式的协议，在传递消息的时候需要自己再增加一层数据的封装格式，使用起来会比WebSocket略麻烦。

另外WebTransport 支持不可靠的UDP发送，这个扩宽了新的场景，这个是WebSocket所不能对比的。  相信WebTransport在成熟之后会抢占WebSocket的一部分使用场景。

## WebTransport 功能列表

WebTransport over HTTP3 提供了以下功能特性：

- 单向流 unidirectional streams
- 双向流 bidirectional streams
- 数据包 datagrams

以上功能均可由任意一端发起。Session ID 可用于解复用 Streams 和 Datagrams 以判断属于不同的 WebTransport session，在网络中，session IDs 使用的是 QUIC 可变长度整数方案编码的 QUIC-TRANSPORT。

## 浏览器WebTransport API

WebTransport 主要提供三种类型的 API

- `datagramReadable`和`datagramWritable`，不可靠数据传输。可用于不需要保证数据发送先后顺序的场景；
- `createBidirectionalStream`，双向数据流可靠传输。可用于需要保证数据发送顺序且关心返回结果的场景；
- `createUnidirectionalStream`，单向数据流可靠传输。可用于需要保证数据发送顺序，但是不关心返回值的场景。


## 感谢

代码基于<https://github.com/marten-seemann/webtransport-go>改造而来。

# 参考资料

- [尝试使用 WebTransport](https://web.dev/i18n/zh/webtransport/)
- [一文看懂WebTransport](https://segmentfault.com/a/1190000039710193)
- [聊聊WebTransport](http://blog.asyncoder.com/2021/02/03/%E8%81%8A%E8%81%8AWebTransport/)
- [RFC 9000: QUIC: A UDP-Based Multiplexed and Secure Transport](https://www.rfc-editor.org/rfc/rfc9000.html)
- [WebTransport over HTTP/3](https://www.ietf.org/archive/id/draft-ietf-webtrans-http3-02.html)
- [Hypertext Transfer Protocol Version 3 (HTTP/3)](https://www.ietf.org/archive/id/draft-ietf-quic-http-34.html)
- [WebTransport overview](https://tools.ietf.org/html/draft-vvv-webtransport-overview-01)
- [WebTransport over QUIC](https://datatracker.ietf.org/doc/html/draft-vvv-webtransport-quic)
- [WebTransport over HTTP/3](https://datatracker.ietf.org/doc/html/draft-vvv-webtransport-http3)
- [WebTransport Explainer](https://github.com/w3c/webtransport/blob/main/explainer.md)
- [Experimenting with QUIC and WebTransport in Go](https://centrifugal.github.io/centrifugo/blog/quic_web_transport/)
- [WebTransport 与 WebCodecs 初探](https://cloud.tencent.com/developer/article/1968372)\
- [WebTransport rfc 翻译](https://www.qnrtc.com/2022/04/29/webtransport-rfc-%E7%BF%BB%E8%AF%91/)
- [Public WebTransport Echo Server](https://webtransport.day/)
