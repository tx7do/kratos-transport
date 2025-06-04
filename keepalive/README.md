# 保活服务

保活服务是采用了 [gRPC的健康检查服务](https://grpc.io/docs/guides/health-checking/) 封装的一个服务。

## 为什么需要保活服务？

因为Broker实际上只同MQ进行通讯，而并不与MQ以外的外界交互，所以，注册服务无法知晓其生存状态。所以，我们需要额外的起一个保活服务与注册服务通讯，以期让注册服务得知Broker服务的死活。

## 如何打开或者关闭保活服务？

只需要注入`WithEnableKeepAlive`：

比如：

```go
asynq.NewServer(
    asynq.WithEnableKeepAlive(true),
)
```

## 遇到错误 panic: RegisterInstance err retry3times request failed,err=request return error code 400

产生这个错误的原因，主要是网络不通。

很多时候，是因为服务器具有多张网卡（包括Docker的虚拟网卡），导致程序自动选择的网卡是到达不了注册服务器的。所以，网络不通。

要解决这个问题，我们提供了一个简单的解决方案:

- 设置环境变量`KRATOS_TRANSPORT_KEEPALIVE_INTERFACE`，绑定一个正确的网卡。
- 设置环境变量`KRATOS_TRANSPORT_KEEPALIVE_HOST`，绑定一个正确的IP地址。
