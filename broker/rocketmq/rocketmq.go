package rocketmq

import (
	"github.com/tx7do/kratos-transport/broker"

	rocketmqOption "github.com/tx7do/kratos-transport/broker/rocketmq/option"

	aliyunMQ "github.com/tx7do/kratos-transport/broker/rocketmq/aliyun"
	rocketmqV2 "github.com/tx7do/kratos-transport/broker/rocketmq/rocketmq-client-go"
	rocketmqV5 "github.com/tx7do/kratos-transport/broker/rocketmq/rocketmq-clients"
)

func NewBroker(driverType rocketmqOption.DriverType, opts ...broker.Option) broker.Broker {

	switch driverType {
	case rocketmqOption.DriverTypeAliyun:
		return aliyunMQ.NewBroker(opts...)
	case rocketmqOption.DriverTypeV2:
		return rocketmqV2.NewBroker(opts...)
	case rocketmqOption.DriverTypeV5:
		return rocketmqV5.NewBroker(opts...)
	default:
		return nil
	}
}
