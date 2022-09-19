# Machinery

Machinery一个第三方开源的基于分布式消息分发的异步任务队列。类似于Python的Celery。

## 特性

- 任务重试机制
- 延迟任务支持
- 任务回调机制
- 任务结果记录
- 支持Workflow模式：Chain，Group，Chord
- 多Brokers支持：Redis, AMQP, AWS SQS
- 多Backends支持：Redis, Memcache, AMQP, MongoDB

## 架构

任务队列，简而言之就是一个放大的生产者消费者模型，用户请求会生成任务，任务生产者不断的向队列中插入任务，同时，队列的处理器程序充当消费者不断的消费任务。

- Server ：业务主体，我们可以使用用server暴露的接口方法进行所有任务编排的操作。如果是简单的使用那么了解它就够了。
- Broker ：数据存储层接口，主要功能是将数据放入任务队列和取出，控制任务并发，延迟也在这层。
- Backend：数据存储层接口，主要用于更新获取任务执行结果，状态等。
- Worker：数据处理层结构，主要是操作 Server、Broker、Backend 进行任务的获取，执行，处理执行状态及结果等。
- Task： 数据处理层，这一层包括Task、Signature、Group、Chain、Chord等结构，主要是处理任务编排的逻辑。

## 任务编排

Machinery一共提供了三种任务编排方式：

- Groups： 执行一组异步任务，任务之间互不影响。
- Chords： 先执行一组同步任务，执行完成后，再调用最后一个回调函数。
- Chains： 执行一组同步任务，任务有次序之分，上个任务的出参可作为下个任务的入参。

## Docker部署开发环境

### Redis

```shell
docker pull bitnami/redis:latest
docker pull bitnami/redis-exporter:latest

docker run -itd \
    --name redis-test \
    -p 6379:6379 \
    -e ALLOW_EMPTY_PASSWORD=yes \
    bitnami/redis:latest
```

## 参考资料

* [Machinery Github](https://github.com/RichardKnop/machinery)
* [machinery中文文档](https://zhuanlan.zhihu.com/p/270640260)
* [Go 语言分布式任务处理器 Machinery – 架构，源码详解篇](https://marksuper.xyz/2022/04/20/machinery1/)
* [Task orchestration in Go Machinery.](https://medium.com/swlh/task-orchestration-in-go-machinery-66a0ddcda548)
* [go-machinery入门教程（异步任务队列）](https://juejin.cn/post/6889743612267986958)
