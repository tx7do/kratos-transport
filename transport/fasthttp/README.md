# FastHTTP

## 什么是FastHTTP?

FastHTTP是golang下的一个http框架，顾名思义，与原生的http实现相比，它的特点在于快，按照官网的说法，它的客户端和服务端性能比原生有了十倍的提升。

它的高性能主要源自于“复用”，通过服务协程和内存变量的复用，节省了大量资源分配的成本。

## 参考资料

- [FastHTTP Github](https://github.com/valyala/fasthttp)
- [fasthttp：高性能背后的惨痛代价](https://cloud.tencent.com/developer/news/462918)
- [fasthttp性能真的比标准库http包好很多吗？一文告诉你真相！](https://zhuanlan.zhihu.com/p/367927669)
