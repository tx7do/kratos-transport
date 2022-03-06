# Redis

利用了Redis的发布订阅命令

## PUBLISH channel message

用于将信息发送到指定的频道。

## SUBSCRIBE channel [channel …]

订阅给定的一个或多个频道的信息

## UNSUBSCRIBE [channel [channel …]]

指退订给定的频道

## PSUBSCRIBE pattern [pattern ...]

订阅一个或多个符合给定模式的频道

## PUBSUB subcommand [argument [argument ...]]

查看订阅与发布系统状态

## PUNSUBSCRIBE [pattern [pattern ...]]

退订所有给定模式的频道
