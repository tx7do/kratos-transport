# SignalR

## 什么是 SignalR？

ASP.NET SignalR 是一个面向 ASP.NET 开发人员的库，可简化向应用程序添加实时 Web 功能的过程。 实时 Web 功能是让服务器代码在可用时立即将内容推送到连接的客户端，而不是让服务器等待客户端请求新数据。

SignalR 可用于向 ASP.NET 应用程序添加任何类型的“实时”Web 功能。 虽然聊天通常用作示例，但你可以执行更多操作。 每当用户刷新网页以查看新数据，或页面实现 长时间轮询 以检索新数据时，它都是使用 SignalR 的候选项。 示例包括仪表板和监视应用程序、协作应用程序 (，例如同时编辑文档) 、作业进度更新和实时表单。

SignalR 还支持需要服务器进行高频率更新的全新 Web 应用程序类型，例如实时游戏。

SignalR 提供了一个简单的 API，用于创建服务器到客户端远程过程调用， (RPC) 调用客户端浏览器 (和其他客户端平台中的 JavaScript 函数，) 从服务器端 .NET 代码。 SignalR 还包括用于连接管理的 API (例如，连接和断开连接事件) ，以及分组连接。

## SignalR 和 WebSocket

SignalR 在可用的情况下使用新的 WebSocket 传输，并在必要时回退到旧传输。 虽然当然可以直接使用 WebSocket 编写应用，但使用 SignalR 意味着需要实现的许多额外功能已经为你完成。 最重要的是，这意味着你可以编写应用代码以利用 WebSocket，而无需担心为旧客户端创建单独的代码路径。 SignalR 还可以避免担心 WebSocket 的更新，因为 SignalR 已更新以支持基础传输中的更改，从而为应用程序提供跨 WebSocket 版本的一致接口。

## 参考资料 (Reference)

- [Introduction to SignalR](https://learn.microsoft.com/en-us/aspnet/signalr/overview/getting-started/introduction-to-signalr)
- [go-signalr](https://github.com/philippseith/signalr)
- [SignalR vs. Socket.IO: which one is best for you?](https://ably.com/topic/signalr-vs-socketio)
