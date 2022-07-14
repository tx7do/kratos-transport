# pulsar

## Docker部署开发环境

```shell
docker pull apachepulsar/pulsar:latest

docker run -itd \
    -p 6650:6650 \
    -p 8080:8080 \
    --name pulsar-standalone \
    apachepulsar/pulsar:latest bin/pulsar standalone
```

```shell
docker pull apachepulsar/pulsar-manager:latest

docker run -itd \
    -p 9527:9527 \
    -p 7750:7750 \
    --name pulsar-manager \
    -e SPRING_CONFIGURATION_FILE=/pulsar-manager/pulsar-manager/application.properties \
    apachepulsar/pulsar-manager:latest
```

设置管理员账号密码：

```shell
CSRF_TOKEN=$(curl http://localhost:7750/pulsar-manager/csrf-token)
curl \
   -H 'X-XSRF-TOKEN: $CSRF_TOKEN' \
   -H 'Cookie: XSRF-TOKEN=$CSRF_TOKEN;' \
   -H "Content-Type: application/json" \
   -X PUT http://localhost:7750/pulsar-manager/users/superuser \
   -d '{"name": "admin", "password": "apachepulsar", "description": "test", 
        "email": "username@test.org"}'
```

管理后台 <http://localhost:9527>  
登陆账号：admin，密码：apachepulsar

在后台创建一个环境，其中，服务地址为pulsar的地址：
* 环境名（Environment Name）： default
* 服务地址（Service URL）：http://host.docker.internal:8080
