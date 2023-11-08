package rocketmqOption

const (
	DefaultAddr = "127.0.0.1:9876"
)

type DriverType string

const (
	DriverTypeAliyun DriverType = "aliyun" // github.com/aliyunmq/mq-http-go-sdk
	DriverTypeV2     DriverType = "v2"     // github.com/apache/rocketmq-client-go/v2
	DriverTypeV5     DriverType = "v5"     // github.com/apache/rocketmq-clients/golang
)

type MessageModel string

const (
	MessageModelBroadCasting MessageModel = "BroadCasting"
	MessageModelClustering   MessageModel = "Clustering"
)
