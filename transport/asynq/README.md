# Asynq

Asynq是一个go语言实现的分布式任务队列和异步处理库，基于Redis。类似于Python的Celery。作者Ken Hibino，任职于Google。

## 特点

- 保证至少执行一次任务
- 任务写入Redis后可以持久化
- 任务失败之后，会自动重试
- worker崩溃自动恢复
- 可是实现任务的优先级
- 任务可以进行编排
- 任务可以设定执行时间或者最长可执行的时间
- 支持中间件
- 可以使用 unique-option 来避免任务重复执行，实现唯一性
- 支持 Redis Cluster 和 Redis Sentinels 以达成高可用性
- 作者提供了Web UI & CLI Tool让大家查看任务的执行情况

## 安装命令行工具

```shell
go install github.com/hibiken/asynq/tools/asynq
```

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

### Asynqmon

```shell
docker pull hibiken/asynqmon:latest

docker run -d \
    --name asynq \
    -p 8080:8080 \
    hibiken/asynqmon:latest --redis-addr=host.docker.internal:6379
```

- 管理后台：<http://localhost:8080>

## 参考资料

* [Asynq Github](https://github.com/hibiken/asynq)
* [Asynq: simple, reliable & efficient distributed task queue for your next Go project](https://dev.to/koddr/asynq-simple-reliable-efficient-distributed-task-queue-for-your-next-go-project-4jhg)
* [Asynq: Golang distributed task queue library](https://nickest14.medium.com/asynq-golang-distributed-task-queue-library-75de3424a830)
