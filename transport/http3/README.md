# HTTP3

## QUIC 协议

QUIC（读作“quick”）是一个通用的传输层网络协议，最初由Google的Jim Roskind设计。该协议于2012年实现并部署，2013年随着实验范围的扩大而公开发布，并向IETF描述。虽然长期处于互联网草案阶段，但从Chrome浏览器至Google服务器的连接中超过一半的连接都使用了QUIC。Microsoft Edge、Firefox都已支持此协议；Safari实现了QUIC，但默认情况下没有启用。QUIC于RFC9000中被正式标准化。

虽然QUIC的名称最初是“快速UDP互联网连接”（Fast UDP Internet Connection）的首字母缩写，但IETF指定的标准中QUIC并不是任何内容的缩写。QUIC提高了目前使用TCP的面向连接的网络应用的性能。它通过使用用户数据报协议（UDP）在两个端点之间创建若干个多路连接来实现这一目标，其目的是为了在网络层淘汰TCP，以满足许多应用的需求，因此该协议偶尔也会获得 “TCP/2”的昵称。

QUIC与HTTP/2的多路复用连接协同工作，允许多个数据流独立到达所有端点，因此不受涉及其他数据流的丢包影响。相反，HTTP/2创建在传输控制协议（TCP）上，如果任何一个TCP数据包延迟或丢失，所有多路数据流都会遭受队头阻塞延迟。

QUIC的次要目标包括降低连接和传输时延，以及每个方向的带宽估计以避免拥塞。它还将拥塞控制算法移到了两个端点的用户空间，而不是内核空间，据称这将使这些算法得到更快的改进。此外，该协议还可以扩展前向纠错（FEC），以进一步提高预期错误时的性能，这被视为协议演进的下一步。

2015年6月，QUIC规范的互联网草案提交给IETF进行标准化。2016年，成立了QUIC工作组。2018年10月，IETF的HTTP工作组和QUIC工作组共同决定将QUIC上的HTTP映射称为 "HTTP/3"，以提前使其成为全球标准。2021年5月IETF公布RFC9000，QUIC规范推出了标准化版本。

## HTTP3

HTTP/3是第三个主要版本的HTTP协议。与其前任HTTP/1.1和HTTP/2不同，在HTTP/3中，将弃用TCP协议，改为使用基于UDP协议的QUIC协议实现。

此变化主要为了解决HTTP/2中存在的队头阻塞问题。由于HTTP/2在单个TCP连接上使用了多路复用，受到TCP拥塞控制的影响，少量的丢包就可能导致整个TCP连接上的所有流被阻塞。

QUIC（快速UDP网络连接）是一种实验性的网络传输协议，由Google开发，该协议旨在使网页传输更快。在2018年10月28日的邮件列表讨论中，互联网工程任务组（IETF） HTTP和QUIC工作组主席Mark Nottingham提出了将HTTP-over-QUIC更名为HTTP/3的正式请求，以“明确地将其标识为HTTP语义的另一个绑定……使人们理解它与QUIC的不同”，并在最终确定并发布草案后，将QUIC工作组继承到HTTP工作组。在随后的几天讨论中，Mark Nottingham的提议得到了IETF成员的接受，他们在2018年11月给出了官方批准，认可HTTP-over-QUIC成为HTTP/3。

2019年9月，HTTP/3支持已添加到Cloudflare和Google Chrome（Canary build）。Firefox Nightly在2019年秋季之后添加支持。

2022年6月6日，IETF正式标准化HTTP/3为RFC9114。

## 参考资料

- [QUIC协议 - 维基百科](https://zh.wikipedia.org/wiki/QUIC)
- [HTTP/3 - 维基百科](https://zh.wikipedia.org/wiki/HTTP/3)
- [深入剖析HTTP3协议](https://www.taohui.tech/2021/02/04/%E7%BD%91%E7%BB%9C%E5%8D%8F%E8%AE%AE/%E6%B7%B1%E5%85%A5%E5%89%96%E6%9E%90HTTP3%E5%8D%8F%E8%AE%AE/)
- [降低20%链路耗时，Trip.com APP QUIC应用和优化实践](https://www.51cto.com/article/706585.html)
- [QUIC 在微博中的落地思考](https://www.infoq.cn/article/2018/03/weibo-quic)

