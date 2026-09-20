package sqs

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"github.com/tx7do/kratos-transport/broker"
)

const (
	defaultAddr   = "http://127.0.0.1:9324"
	defaultRegion = "us-east-1"
)

type sqsBroker struct {
	sync.RWMutex

	options broker.Options

	client *sqs.Client

	region   string
	endpoint string

	running bool

	subscribers *broker.SubscriberSyncMap

	queueUrlMu    sync.Mutex
	queueUrlCache map[string]string
}

func NewBroker(opts ...broker.Option) broker.Broker {
	options := broker.NewOptionsAndApply(opts...)

	b := &sqsBroker{
		options:       options,
		subscribers:   broker.NewSubscriberSyncMap(),
		queueUrlCache: make(map[string]string),
	}

	return b
}

func (b *sqsBroker) Name() string {
	return "SQS"
}

func (b *sqsBroker) Options() broker.Options {
	return b.options
}

func (b *sqsBroker) Address() string {
	if len(b.options.Addrs) > 0 {
		return b.options.Addrs[0]
	}
	return defaultAddr
}

func (b *sqsBroker) Init(opts ...broker.Option) error {
	for _, o := range opts {
		o(&b.options)
	}

	b.configure(b.options.Context)

	return nil
}

func (b *sqsBroker) configure(ctx context.Context) {
	if ctx == nil {
		return
	}

	if v, ok := ctx.Value(regionKey{}).(string); ok && v != "" {
		b.region = v
	}
	if v, ok := ctx.Value(endpointKey{}).(string); ok && v != "" {
		b.endpoint = v
	}
}

func (b *sqsBroker) Connect() error {
	b.Lock()
	defer b.Unlock()

	if b.running {
		return nil
	}

	region := b.region
	if region == "" {
		region = defaultRegion
	}

	var opts []func(*config.LoadOptions) error

	opts = append(opts, config.WithRegion(region))

	if b.endpoint != "" {
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...any) (aws.Endpoint, error) {
			if service == sqs.ServiceID {
				return aws.Endpoint{
					URL:           b.endpoint,
					SigningRegion: region,
				}, nil
			}
			return aws.Endpoint{}, fmt.Errorf("unknown endpoint for service: %s, region: %s", service, region)
		})
		opts = append(opts, config.WithEndpointResolverWithOptions(customResolver))
	}

	cfg, err := config.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	b.client = sqs.NewFromConfig(cfg)

	b.running = true

	LogInfof("connected to SQS, region: %s, endpoint: %s", region, b.endpoint)

	return nil
}

func (b *sqsBroker) Disconnect() error {
	b.Lock()
	defer b.Unlock()

	if !b.running {
		return nil
	}

	b.subscribers.Clear()

	b.client = nil
	b.running = false

	LogInfo("disconnected from SQS")

	return nil
}

func (b *sqsBroker) Request(ctx context.Context, topic string, msg *broker.Message, opts ...broker.RequestOption) (*broker.Message, error) {
	return broker.GenericRequest(ctx, b, topic, msg, opts...)
}

func (b *sqsBroker) Publish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	var finalTask = b.internalPublish

	if len(b.options.PublishMiddlewares) > 0 {
		finalTask = broker.ChainPublishMiddleware(finalTask, b.options.PublishMiddlewares)
	}

	return finalTask(ctx, topic, msg, opts...)
}

func (b *sqsBroker) internalPublish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	buf, err := broker.Marshal(b.options.Codec, msg.Body)
	if err != nil {
		return err
	}

	sendMsg := msg.Clone()
	sendMsg.Body = buf

	return b.publish(ctx, topic, sendMsg, opts...)
}

func (b *sqsBroker) publish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	if b.client == nil {
		return errors.New("SQS client is nil")
	}

	options := broker.PublishOptions{
		Context: ctx,
	}
	for _, o := range opts {
		o(&options)
	}

	queueUrl := b.resolveQueueUrl(options.Context, topic)
	if queueUrl == "" {
		return fmt.Errorf("queue url not resolved for topic: %s", topic)
	}

	input := &sqs.SendMessageInput{
		QueueUrl:    &queueUrl,
		MessageBody: aws.String(string(msg.BodyBytes())),
	}

	// Extract publish options
	if options.Context != nil {
		if v, ok := options.Context.Value(delaySecondsKey{}).(int32); ok && v > 0 {
			input.DelaySeconds = v
		}
		if v, ok := options.Context.Value(messageGroupIdKey{}).(string); ok && v != "" {
			input.MessageGroupId = &v
		}
		if v, ok := options.Context.Value(messageDeduplicationIdKey{}).(string); ok && v != "" {
			input.MessageDeduplicationId = &v
		}
	}

	// Set message attributes from headers
	if msg.Headers != nil {
		attrs := make(map[string]types.MessageAttributeValue)
		for k, v := range msg.Headers {
			attrs[k] = types.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: aws.String(v),
			}
		}
		if len(attrs) > 0 {
			input.MessageAttributes = attrs
		}
	}

	_, err := b.client.SendMessage(ctx, input)
	if err != nil {
		return fmt.Errorf("send message failed: %w", err)
	}

	return nil
}

func (b *sqsBroker) Subscribe(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	if b.client == nil {
		return nil, errors.New("SQS client is nil, call Connect() first")
	}

	options := broker.SubscribeOptions{
		Context: context.Background(),
		AutoAck: true,
	}
	for _, o := range opts {
		o(&options)
	}

	if len(b.options.SubscriberMiddlewares) > 0 {
		handler = broker.ChainSubscriberMiddleware(handler, b.options.SubscriberMiddlewares)
	}

	queueUrl := b.resolveQueueUrl(options.Context, topic)
	if queueUrl == "" {
		return nil, fmt.Errorf("queue url not resolved for topic: %s", topic)
	}

	// Extract subscribe options
	visibilityTimeout := int32(DefaultVisibilityTimeout)
	waitTimeSeconds := int32(DefaultWaitTimeSeconds)
	maxMessages := int32(DefaultMaxMessages)

	if options.Context != nil {
		if v, ok := options.Context.Value(visibilityTimeoutKey{}).(int32); ok && v > 0 {
			visibilityTimeout = v
		}
		if v, ok := options.Context.Value(waitTimeSecondsKey{}).(int32); ok && v > 0 {
			waitTimeSeconds = v
		}
		if v, ok := options.Context.Value(maxMessagesKey{}).(int32); ok && v > 0 {
			maxMessages = v
		}
	}

	sub := &subscriber{
		topic:    topic,
		queueUrl: queueUrl,
		options:  options,
		b:        b,
		client:   b.client,
	}

	// 在启动 goroutine 之前创建并登记 cancel，
	// 避免「Subscribe 后立刻 Unsubscribe」时 cancel 尚未赋值导致消费循环无法停止
	recvCtx, cancel := context.WithCancel(options.Context)
	sub.cancel = cancel

	go sub.recv(recvCtx, handler, binder, recvOpts{
		visibilityTimeout: visibilityTimeout,
		waitTimeSeconds:   waitTimeSeconds,
		maxMessages:       maxMessages,
	})

	if old := b.subscribers.Get(topic); old != nil {
		// 同主题重复订阅：先退订旧订阅，避免旧订阅继续消费（泄漏 + 重复消费）
		if uerr := old.Unsubscribe(false); uerr != nil {
			LogWarnf("unsubscribe old subscriber for topic %q failed: %v", topic, uerr)
		}
	}
	b.subscribers.Add(topic, sub)

	return sub, nil
}

// resolveQueueUrl resolves the queue URL for a given topic.
// Priority: subscribe/publish context > broker option context > derive from topic name
func (b *sqsBroker) resolveQueueUrl(ctx context.Context, topic string) string {
	if ctx != nil {
		if v, ok := ctx.Value(queueUrlKey{}).(string); ok && v != "" {
			return v
		}
	}
	if b.options.Context != nil {
		if v, ok := b.options.Context.Value(queueUrlKey{}).(string); ok && v != "" {
			return v
		}
	}

	// Try to get queue URL from SQS by topic name（按 topic 缓存，避免每次发布一次 API 调用）
	b.queueUrlMu.Lock()
	if cached, ok := b.queueUrlCache[topic]; ok {
		b.queueUrlMu.Unlock()
		return cached
	}
	b.queueUrlMu.Unlock()

	if b.client != nil {
		result, err := b.client.GetQueueUrl(context.Background(), &sqs.GetQueueUrlInput{
			QueueName: &topic,
		})
		if err == nil && result.QueueUrl != nil {
			b.queueUrlMu.Lock()
			b.queueUrlCache[topic] = *result.QueueUrl
			b.queueUrlMu.Unlock()
			return *result.QueueUrl
		}
		LogWarnf("failed to get queue url for %s: %v", topic, err)
	}

	return ""
}
