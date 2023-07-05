# SignalR

SignalR 简化了向应用添加实时 Web 功能的过程。 实时 Web 功能使服务器端代码能够即时将内容推送到客户端。
SignalR 的适用对象：

- 需要来自服务器的高频率更新的应用。 例如：游戏、社交网络、投票、拍卖、地图和 GPS 应用。
- 仪表板和监视应用。 示例包括公司仪表板、销售状态即时更新或行程警示。
- 协作应用。 协作应用的示例包括白板应用和团队会议软件。
- 需要通知的应用。 社交网络、电子邮件、聊天、游戏、行程警示以及许多其他应用都使用通知。

## 什么是 SignalR？

ASP.NET SignalR 是一个面向 ASP.NET 开发人员的库，可简化向应用程序添加实时 Web 功能的过程。 实时 Web 功能是让服务器代码在可用时立即将内容推送到连接的客户端，而不是让服务器等待客户端请求新数据。

SignalR 可用于向 ASP.NET 应用程序添加任何类型的“实时”Web 功能。 虽然聊天通常用作示例，但你可以执行更多操作。 每当用户刷新网页以查看新数据，或页面实现 长时间轮询 以检索新数据时，它都是使用 SignalR 的候选项。 示例包括仪表板和监视应用程序、协作应用程序 (，例如同时编辑文档) 、作业进度更新和实时表单。

SignalR 还支持需要服务器进行高频率更新的全新 Web 应用程序类型，例如实时游戏。

SignalR 提供了一个简单的 API，用于创建服务器到客户端远程过程调用， (RPC) 调用客户端浏览器 (和其他客户端平台中的 JavaScript 函数，) 从服务器端 .NET 代码。 SignalR 还包括用于连接管理的 API (例如，连接和断开连接事件) ，以及分组连接。

## SignalR 和 WebSocket

SignalR 在可用的情况下使用新的 WebSocket 传输，并在必要时回退到旧传输。 虽然当然可以直接使用 WebSocket 编写应用，但使用 SignalR 意味着需要实现的许多额外功能已经为你完成。 最重要的是，这意味着你可以编写应用代码以利用 WebSocket，而无需担心为旧客户端创建单独的代码路径。 SignalR 还可以避免担心 WebSocket 的更新，因为 SignalR 已更新以支持基础传输中的更改，从而为应用程序提供跨 WebSocket 版本的一致接口。

## 传输方式

协商传输需要一定的时间和客户端/服务器资源。如果客户端能力已知，则可以在客户端连接启动时指定传输。以下代码片段演示了使用Ajax Long Polling传输启动连接，如果知道客户端不支持任何其他协议，则使用该代码：

```js
connection.start({ transport: 'longPolling' });
```

如果您希望客户端按顺序尝试特定传输，则可以指定回退顺序。以下代码片段演示了尝试WebSocket，并且失败，直接转到长轮询。

```js
connection.start({ transport: ['webSockets','longPolling'] });
```

它支持的传输方式：

- webSockets
- foreverFrame
- serverSentEvents
- longPolling


## 参考资料 (Reference)

- [Introduction to SignalR](https://learn.microsoft.com/en-us/aspnet/signalr/overview/getting-started/introduction-to-signalr)
- [go-signalr](https://github.com/philippseith/signalr)
- [SignalR vs. Socket.IO: which one is best for you?](https://ably.com/topic/signalr-vs-socketio)
- [SignalR 从开发到生产部署闭坑指南](https://juejin.cn/post/7021724750942568456)
- [ASP.NET - SignalR 理解](https://hackmd.io/@Robin98727/HJ5m4ZJwB)
