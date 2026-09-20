package gcpubsub

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"cloud.google.com/go/pubsub/v2"
	"google.golang.org/api/option"

	"github.com/tx7do/kratos-transport/broker"
	"time"
)

type gcpBroker struct {
	sync.RWMutex

	options broker.Options

	client *pubsub.Client

	running bool

	subscribers *broker.SubscriberSyncMap

	publishersMu sync.Mutex
	publishers   map[string]*pubsub.Publisher
}

func NewBroker(opts ...broker.Option) broker.Broker {
	options := broker.NewOptionsAndApply(opts...)

	b := &gcpBroker{
		options:     options,
		publishers:  make(map[string]*pubsub.Publisher),
		subscribers: broker.NewSubscriberSyncMap(),
	}

	return b
}

func (b *gcpBroker) Name() string {
	return "gcpubsub"
}

func (b *gcpBroker) Options() broker.Options {
	return b.options
}

func (b *gcpBroker) Address() string {
	if len(b.options.Addrs) > 0 {
		return b.options.Addrs[0]
	}
	return ""
}

func (b *gcpBroker) Init(opts ...broker.Option) error {
	for _, o := range opts {
		o(&b.options)
	}
	return nil
}

func (b *gcpBroker) Connect() error {
	b.Lock()
	defer b.Unlock()

	if b.running {
		return nil
	}

	ctx := context.Background()

	projectID := ""
	if b.options.Context != nil {
		if v, ok := b.options.Context.Value(projectIDKey{}).(string); ok && v != "" {
			projectID = v
		}
	}

	if projectID == "" {
		return errors.New("gcp project id is required, use WithProjectID() to set it")
	}

	var clientOpts []option.ClientOption

	if b.options.Context != nil {
		if v, ok := b.options.Context.Value(credentialsFileKey{}).(string); ok && v != "" {
			clientOpts = append(clientOpts, option.WithCredentialsFile(v))
		}
		if v, ok := b.options.Context.Value(endpointKey{}).(string); ok && v != "" {
			clientOpts = append(clientOpts, option.WithEndpoint(v))
		}
	}

	client, err := pubsub.NewClient(ctx, projectID, clientOpts...)
	if err != nil {
		return fmt.Errorf("create pubsub client error: %w", err)
	}

	b.client = client
	b.running = true

	LogInfof("connected to GCP Pub/Sub, project: %s", projectID)

	return nil
}

func (b *gcpBroker) Disconnect() error {
	b.publishersMu.Lock()
	for _, p := range b.publishers {
		p.Stop()
	}
	b.publishers = make(map[string]*pubsub.Publisher)
	b.publishersMu.Unlock()

	b.Lock()
	defer b.Unlock()

	if !b.running {
		return nil
	}

	b.subscribers.Clear()

	if b.client != nil {
		_ = b.client.Close()
	}

	b.client = nil
	b.running = false

	LogInfo("disconnected from GCP Pub/Sub")

	return nil
}

func (b *gcpBroker) Request(ctx context.Context, topic string, msg *broker.Message, opts ...broker.RequestOption) (*broker.Message, error) {
	return broker.GenericRequest(ctx, b, topic, msg, opts...)
}

func (b *gcpBroker) Publish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	var finalTask broker.PublishHandler = b.internalPublish

	if len(b.options.PublishMiddlewares) > 0 {
		finalTask = broker.ChainPublishMiddleware(finalTask, b.options.PublishMiddlewares)
	}

	return finalTask(ctx, topic, msg, opts...)
}

func (b *gcpBroker) internalPublish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	publishOpts := broker.NewPublishOptions(opts...)
	// WithPublishTimeout 接线：为本次发布加超时
	if publishOpts.Context.Value(publishTimeoutKey{}) != nil {
		if d, ok := publishOpts.Context.Value(publishTimeoutKey{}).(time.Duration); ok && d > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, d)
			defer cancel()
		}
	}

	buf, err := broker.Marshal(b.options.Codec, msg.Body)
	if err != nil {
		return err
	}

	sendMsg := msg.Clone()
	sendMsg.Body = buf

	return b.publish(ctx, topic, sendMsg, opts...)
}

func (b *gcpBroker) publish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	b.RLock()
	client := b.client
	b.RUnlock()

	if client == nil {
		return errors.New("GCP Pub/Sub client is nil")
	}

	options := broker.PublishOptions{
		Context: ctx,
	}
	for _, o := range opts {
		o(&options)
	}

	b.publishersMu.Lock()
	if b.publishers == nil {
		b.publishers = make(map[string]*pubsub.Publisher)
	}
	t, ok := b.publishers[topic]
	if !ok {
		// Publisher 携带后台 bundler goroutine，必须复用并在 Disconnect 时 Stop，
		// 否则每次发布都泄漏一组 goroutine
		t = client.Publisher(topic)
		b.publishers[topic] = t
	}
	b.publishersMu.Unlock()

	pubsubMsg := &pubsub.Message{
		Data: msg.BodyBytes(),
	}

	if msg.Headers != nil {
		attrs := make(map[string]string, len(msg.Headers))
		for k, v := range msg.Headers {
			attrs[k] = v
		}
		pubsubMsg.Attributes = attrs
	}

	if options.Context != nil {
		if v, ok := options.Context.Value(publishOrderingKey{}).(string); ok && v != "" {
			pubsubMsg.OrderingKey = v
			// 库要求显式开启，否则带 OrderingKey 的发布必然失败
			t.EnableMessageOrdering = true
		}
	}

	result := t.Publish(ctx, pubsubMsg)

	_, err := result.Get(ctx)
	if err != nil {
		return fmt.Errorf("publish message error: %w", err)
	}

	return nil
}

func (b *gcpBroker) Subscribe(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	b.RLock()
	client := b.client
	b.RUnlock()

	if client == nil {
		return nil, errors.New("GCP Pub/Sub client is nil, call Connect() first")
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

	// Resolve subscription name: subscribe context → topic name as default
	subscriptionName := topic
	if options.Context != nil {
		if v, ok := options.Context.Value(subscriptionNameKey{}).(string); ok && v != "" {
			subscriptionName = v
		}
	}

	var receiveSettings pubsub.ReceiveSettings
	if options.Context != nil {
		if v, ok := options.Context.Value(receiveSettingsKey{}).(pubsub.ReceiveSettings); ok {
			receiveSettings = v
		}
	}

	sub := &subscriber{
		topic:   topic,
		options: options,
		b:       b,
	}

	// 在启动 goroutine 之前创建并登记 cancel，
	// 避免「Subscribe 后立刻 Unsubscribe」时 cancel 尚未赋值导致 Receive 无法停止
	recvCtx, cancel := context.WithCancel(options.Context)
	sub.cancel = cancel

	go b.receive(recvCtx, client, subscriptionName, receiveSettings, handler, binder, options, sub)

	if old := b.subscribers.Get(topic); old != nil {
		// 同主题重复订阅：先退订旧订阅，避免旧订阅继续消费（泄漏 + 重复消费）
		if uerr := old.Unsubscribe(false); uerr != nil {
			LogWarnf("unsubscribe old subscriber for topic %q failed: %v", topic, uerr)
		}
	}
	b.subscribers.Add(topic, sub)

	return sub, nil
}

func (b *gcpBroker) receive(ctx context.Context, client *pubsub.Client, subscriptionName string,
	receiveSettings pubsub.ReceiveSettings, handler broker.Handler, binder broker.Binder,
	options broker.SubscribeOptions, sub *subscriber) {

	subClient := client.Subscriber(subscriptionName)
	subClient.ReceiveSettings = receiveSettings

	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	defer func() {
		LogInfof("subscriber stopped, topic: %s, subscription: %s", sub.topic, subscriptionName)
	}()

	err := subClient.Receive(subCtx, func(receiveCtx context.Context, msg *pubsub.Message) {
		handlePubSubMessage(receiveCtx, b, msg, handler, binder, options, sub)
	})

	// 致命错误（订阅被删、权限问题等）后不能静默死亡：
	// 带退避循环重试，直到订阅被显式关闭或上下文取消（不用递归，避免长故障下栈增长）
	for err != nil {
		if subCtx.Err() != nil {
			// context cancelled, normal exit
			return
		}
		LogErrorf("receive message error: %v, retrying in 5s...", err)
		select {
		case <-subCtx.Done():
			return
		case <-time.After(5 * time.Second):
		}
		err = subClient.Receive(subCtx, func(receiveCtx context.Context, msg *pubsub.Message) {
			handlePubSubMessage(receiveCtx, b, msg, handler, binder, options, sub)
		})
	}
}

// handlePubSubMessage 单条消息的处理逻辑（从 receive 的闭包提取，供重试循环复用）
func handlePubSubMessage(receiveCtx context.Context, b *gcpBroker, msg *pubsub.Message,
	handler broker.Handler, binder broker.Binder,
	options broker.SubscribeOptions, sub *subscriber) {

	var m broker.Message

	// Extract headers from message attributes
	if msg.Attributes != nil {
		m.Headers = make(broker.Headers)
		for k, v := range msg.Attributes {
			m.Headers[k] = v
		}
	}

	// Extract body
	if len(msg.Data) > 0 {
		if binder != nil {
			m.Body = binder()
			if err := broker.Unmarshal(b.options.Codec, msg.Data, &m.Body); err != nil {
				LogErrorf("unmarshal message failed: %v", err)
				msg.Nack()
				return
			}
		} else {
			m.Body = msg.Data
		}
	}

	p := &publication{
		topic:  sub.topic,
		msg:    &m,
		gcpMsg: msg,
		ack:    msg.Ack,
	}

	if err := handler(receiveCtx, p); err != nil {
		p.err = err
		LogErrorf("handle message failed: %v", err)
		if eh := b.options.ErrorHandler; eh != nil {
			_ = eh(receiveCtx, p)
		}
		// 与 unmarshal 失败路径一致：Nack 触发重投
		msg.Nack()
		return
	}

	if options.AutoAck {
		if err := p.Ack(); err != nil {
			LogErrorf("unable to ack msg: %v", err)
		}
	}
}
