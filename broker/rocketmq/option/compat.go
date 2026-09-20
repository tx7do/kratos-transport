package rocketmqOption

import (
	"context"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
)

// 驱动兼容性说明：
//
// 本包的 Option 由三个 RocketMQ 驱动共享：
//   - aliyun（阿里云 HTTP 网关驱动，broker/rocketmq/aliyun）
//   - v2（rocketmq-client-go，broker/rocketmq/rocketmq-client-go）
//   - v5（rocketmq-clients，broker/rocketmq/rocketmq-clients）
//
// 某个驱动不支持的选项会被静默忽略。为避免"传了不生效"的困惑，
// 各驱动在 Init/Publish/Subscribe 时通过 WarnUnsupportedKeysOnce 对
// 不支持的选项打一次性告警。各选项的支持情况见 With* 函数的注释。

type keySupport struct {
	key  any
	name string
}

var warnedKeys sync.Map // "driver/name" -> struct{}

// WarnUnsupportedKeysOnce 检查 ctx 中出现了哪些本驱动不支持的 key，
// 对每个 driver+key 组合只告警一次。
func WarnUnsupportedKeysOnce(driver string, ctx context.Context, unsupported []keySupport) {
	if ctx == nil {
		return
	}
	for _, k := range unsupported {
		if ctx.Value(k.key) == nil {
			continue
		}
		tag := driver + "/" + k.name
		if _, loaded := warnedKeys.LoadOrStore(tag, struct{}{}); !loaded {
			log.Warnf("[rocketmq/%s] option %q is not supported by this driver and will be ignored", driver, k.name)
		}
	}
}

// AliyunUnsupportedBrokerKeys aliyun HTTP 驱动不支持的 Broker 级选项
func AliyunUnsupportedBrokerKeys() []keySupport {
	return []keySupport{
		{nameEnableTrace, "WithEnableTrace"}, // 无消息轨迹支持
		{nameLoggerLevel, "WithLoggerLevel"}, // HTTP SDK 无日志级别配置
		{nameSubscriptionExpressions, "WithSubscriptionExpressions"},
		{nameAwaitDuration, "WithAwaitDuration"},
		{nameMaxMessageNum, "WithMaxMessageNumKey"},
		{nameInvisibleDuration, "WithInvisibleDuration"},
		{nameReceiveInterval, "WithReceiveInterval"},
		{nameRetryCount, "WithRetryCount"}, // HTTP SDK 不暴露重试配置
	}
}

// AliyunUnsupportedPublishKeys aliyun HTTP 驱动不支持的 Publish 选项
func AliyunUnsupportedPublishKeys() []keySupport {
	return []keySupport{
		{nameCompress, "WithCompress"},
		{nameBatch, "WithBatch"},
		{nameDelayTimeLevel, "WithDelayTimeLevel"}, // StartDeliverTime 是绝对时间戳，见 WithDelayTimeLevel
		{nameMessageGroup, "WithMessageGroup"},
		{nameSendAsync, "WithSendAsync"},
		{nameSendWithTransaction, "WithSendWithTransaction"},
	}
}

// V2UnsupportedBrokerKeys rocketmq-client-go(v2) 驱动不支持的 Broker 级选项
func V2UnsupportedBrokerKeys() []keySupport {
	return []keySupport{
		{nameSubscriptionExpressions, "WithSubscriptionExpressions"},
		{nameAwaitDuration, "WithAwaitDuration"},
		{nameMaxMessageNum, "WithMaxMessageNumKey"},
		{nameInvisibleDuration, "WithInvisibleDuration"},
		{nameReceiveInterval, "WithReceiveInterval"},
	}
}

// V2UnsupportedPublishKeys v2 驱动不支持的 Publish 选项
func V2UnsupportedPublishKeys() []keySupport {
	return []keySupport{
		{nameDeliveryTimestamp, "WithDeliveryTimestamp"}, // v2 仅支持延迟级别（WithDelayTimeLevel）
		{nameMessageGroup, "WithMessageGroup"},
		{nameSendAsync, "WithSendAsync"},
		{nameSendWithTransaction, "WithSendWithTransaction"},
	}
}

// V2UnsupportedSubscribeKeys v2 驱动不支持的 Subscribe 选项
func V2UnsupportedSubscribeKeys() []keySupport {
	return []keySupport{
		{nameSubscriptionFilterExpr, "WithSubscriptionFilterExpression"},
	}
}

// V5UnsupportedBrokerKeys rocketmq-clients(v5) 驱动不支持的 Broker 级选项
func V5UnsupportedBrokerKeys() []keySupport {
	return []keySupport{
		{nameRetryCount, "WithRetryCount"}, // SDK 不暴露发送重试配置
	}
}

// V5UnsupportedPublishKeys v5 驱动不支持的 Publish 选项
func V5UnsupportedPublishKeys() []keySupport {
	return []keySupport{
		{nameCompress, "WithCompress"},
		{nameBatch, "WithBatch"},
		{nameDelayTimeLevel, "WithDelayTimeLevel"}, // v5 使用绝对时间戳（WithDeliveryTimestamp）
		{nameShardingKey, "WithShardingKey"},
	}
}

// V5UnsupportedSubscribeKeys v5 驱动不支持的 Subscribe 选项
func V5UnsupportedSubscribeKeys() []keySupport {
	return []keySupport{
		{nameConsumerModel, "WithConsumerModel"},
	}
}

var (
	nameEnableTrace             = EnableTraceKey{}
	nameNameServers             = NameServersKey{}
	nameNameServerURL           = NameServerUrlKey{}
	nameAccessKey               = AccessKey{}
	nameSecretKey               = SecretKey{}
	nameSecurityToken           = SecurityTokenKey{}
	nameCredentials             = CredentialsKey{}
	nameRetryCount              = RetryCountKey{}
	nameNamespace               = NamespaceKey{}
	nameInstanceName            = InstanceNameKey{}
	nameGroupName               = GroupNameKey{}
	nameSubscriptionExpressions = SubscriptionExpressionsKey{}
	nameAwaitDuration           = AwaitDurationKey{}
	nameMaxMessageNum           = MaxMessageNumKey{}
	nameInvisibleDuration       = InvisibleDurationKey{}
	nameReceiveInterval         = ReceiveIntervalKey{}
	nameLoggerLevel             = LoggerLevelKey{}
	nameCompress                = CompressKey{}
	nameBatch                   = BatchKey{}
	nameProperties              = PropertiesKey{}
	nameDelayTimeLevel          = DelayTimeLevelKey{}
	nameTag                     = TagsKey{}
	nameKeys                    = KeysKey{}
	nameShardingKey             = ShardingKeyKey{}
	nameDeliveryTimestamp       = DeliveryTimestampKey{}
	nameMessageGroup            = MessageGroupKey{}
	nameSendAsync               = SendAsyncKey{}
	nameSendWithTransaction     = SendWithTransactionKey{}
	nameSubscriptionFilterExpr  = SubscriptionFilterExpressionKey{}
	nameConsumerModel           = ConsumerModelKey{}
)
