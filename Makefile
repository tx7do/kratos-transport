APP_VERSION=v1.0.1

	PACKAGE_LIST = broker/kafka/ broker/mqtt/ broker/nats/ broker/nsq/ broker/pulsar/ broker/rabbitmq/ broker/redis/ broker/rocketmq/ broker/stomp/ transport/activemq/ transport/asynq/ transport/fasthttp/ transport/gin/ transport/graphql/ transport/hertz/ transport/http3/ transport/kafka/ transport/machinery/ transport/mqtt/ transport/nats/ transport/nsq/ transport/pulsar/ transport/rabbitmq/ transport/redis/ transport/rocketmq/ transport/thrift/ transport/websocket/ transport/webtransport/

.PHONY: tag
tag:
	git tag -f $(APP_VERSION) && $(foreach item, $(PACKAGE_LIST), git tag -f $(item)$(APP_VERSION) && ) git push --tags --force
