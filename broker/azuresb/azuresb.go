package azuresb

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/admin"

	"github.com/tx7do/kratos-transport/broker"
)

type azureBroker struct {
	sync.RWMutex

	options broker.Options

	client *azservicebus.Client

	running bool

	subscribers *broker.SubscriberSyncMap
}

func NewBroker(opts ...broker.Option) broker.Broker {
	options := broker.NewOptionsAndApply(opts...)

	b := &azureBroker{
		options:     options,
		subscribers: broker.NewSubscriberSyncMap(),
	}

	return b
}

func (b *azureBroker) Name() string {
	return "azuresb"
}

func (b *azureBroker) Options() broker.Options {
	return b.options
}

func (b *azureBroker) Address() string {
	if len(b.options.Addrs) > 0 {
		return b.options.Addrs[0]
	}
	return ""
}

func (b *azureBroker) Init(opts ...broker.Option) error {
	for _, o := range opts {
		o(&b.options)
	}
	return nil
}

func (b *azureBroker) Connect() error {
	b.Lock()
	defer b.Unlock()

	if b.running {
		return nil
	}

	connStr := ""
	if b.options.Context != nil {
		if v, ok := b.options.Context.Value(connectionStringKey{}).(string); ok && v != "" {
			connStr = v
		}
	}

	if connStr == "" {
		return errors.New("connection string is required, use WithConnectionString() to set it")
	}

	client, err := azservicebus.NewClientFromConnectionString(connStr, nil)
	if err != nil {
		return fmt.Errorf("create service bus client error: %w", err)
	}

	b.client = client
	b.running = true

	LogInfo("connected to Azure Service Bus")

	return nil
}

func (b *azureBroker) Disconnect() error {
	b.Lock()
	defer b.Unlock()

	if !b.running {
		return nil
	}

	b.subscribers.Clear()

	if b.client != nil {
		_ = b.client.Close(context.Background())
	}

	b.client = nil
	b.running = false

	LogInfo("disconnected from Azure Service Bus")

	return nil
}

func (b *azureBroker) Request(ctx context.Context, topic string, msg *broker.Message, opts ...broker.RequestOption) (*broker.Message, error) {
	return broker.GenericRequest(ctx, b, topic, msg, opts...)
}

func (b *azureBroker) Publish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	var finalTask broker.PublishHandler = b.internalPublish

	if len(b.options.PublishMiddlewares) > 0 {
		finalTask = broker.ChainPublishMiddleware(finalTask, b.options.PublishMiddlewares)
	}

	return finalTask(ctx, topic, msg, opts...)
}

func (b *azureBroker) internalPublish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	buf, err := broker.Marshal(b.options.Codec, msg.Body)
	if err != nil {
		return err
	}

	sendMsg := msg.Clone()
	sendMsg.Body = buf

	return b.publish(ctx, topic, sendMsg, opts...)
}

func (b *azureBroker) publish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	b.RLock()
	client := b.client
	b.RUnlock()

	if client == nil {
		return errors.New("Azure Service Bus client is nil")
	}

	options := broker.PublishOptions{
		Context: ctx,
	}
	for _, o := range opts {
		o(&options)
	}

	sender, err := client.NewSender(topic, nil)
	if err != nil {
		return fmt.Errorf("create sender error: %w", err)
	}
	defer sender.Close(ctx)

	sbMsg := &azservicebus.Message{
		Body: msg.BodyBytes(),
	}

	if msg.Headers != nil {
		sbMsg.ApplicationProperties = make(map[string]any, len(msg.Headers))
		for k, v := range msg.Headers {
			sbMsg.ApplicationProperties[k] = v
		}
	}

	if options.Context != nil {
		if v, ok := options.Context.Value(publishContentTypeKey{}).(string); ok && v != "" {
			sbMsg.ContentType = &v
		}
		if v, ok := options.Context.Value(publishSessionIDKey{}).(string); ok && v != "" {
			sbMsg.SessionID = &v
		}
		if v, ok := options.Context.Value(publishMessageIDKey{}).(string); ok && v != "" {
			sbMsg.MessageID = &v
		}
	}

	err = sender.SendMessage(ctx, sbMsg, nil)
	if err != nil {
		return fmt.Errorf("send message error: %w", err)
	}

	return nil
}

func (b *azureBroker) Subscribe(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	b.RLock()
	client := b.client
	b.RUnlock()

	if client == nil {
		return nil, errors.New("Azure Service Bus client is nil, call Connect() first")
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

	// Build receiver options
	receiverOpts := &azservicebus.ReceiverOptions{
		ReceiveMode: azservicebus.ReceiveModePeekLock,
	}

	if options.Context != nil {
		if v, ok := options.Context.Value(receiveModeKey{}).(azservicebus.ReceiveMode); ok {
			receiverOpts.ReceiveMode = v
		}
	}

	// Determine if subscribing to a queue or a topic/subscription
	subscriptionName := ""
	if options.Context != nil {
		if v, ok := options.Context.Value(subscriptionNameKey{}).(string); ok && v != "" {
			subscriptionName = v
		}
	}

	var receiver *azservicebus.Receiver
	var err error

	if subscriptionName != "" {
		// Topic subscription
		receiver, err = client.NewReceiverForSubscription(topic, subscriptionName, receiverOpts)
	} else {
		// Queue
		receiver, err = client.NewReceiverForQueue(topic, receiverOpts)
	}

	if err != nil {
		return nil, fmt.Errorf("create receiver error: %w", err)
	}

	sub := &subscriber{
		topic:   topic,
		options: options,
		b:       b,
	}

	subCtx, cancel := context.WithCancel(options.Context)
	sub.Lock()
	sub.cancel = cancel
	sub.Unlock()

	go b.receive(subCtx, receiver, handler, binder, options, sub)

	b.subscribers.Add(topic, sub)

	return sub, nil
}

func (b *azureBroker) receive(ctx context.Context, receiver *azservicebus.Receiver,
	handler broker.Handler, binder broker.Binder, options broker.SubscribeOptions,
	sub *subscriber) {

	defer func() {
		LogInfof("subscriber stopped, topic: %s", sub.topic)
		_ = receiver.Close(ctx)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		messages, err := receiver.ReceiveMessages(ctx, 1, nil)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			LogErrorf("receive messages error: %v", err)
			continue
		}

		for _, msg := range messages {
			b.processMessage(ctx, receiver, handler, binder, options, sub, msg)
		}
	}
}

func (b *azureBroker) processMessage(ctx context.Context, receiver *azservicebus.Receiver,
	handler broker.Handler, binder broker.Binder, options broker.SubscribeOptions,
	sub *subscriber, sbMsg *azservicebus.ReceivedMessage) {

	var m broker.Message

	// Extract headers from application properties
	if sbMsg.ApplicationProperties != nil {
		m.Headers = make(broker.Headers)
		for k, v := range sbMsg.ApplicationProperties {
			if s, ok := v.(string); ok {
				m.Headers[k] = s
			}
		}
	}

	// Extract body
	if len(sbMsg.Body) > 0 {
		if binder != nil {
			m.Body = binder()
			if err := broker.Unmarshal(b.options.Codec, sbMsg.Body, &m.Body); err != nil {
				LogErrorf("unmarshal message failed: %v", err)
				_ = receiver.AbandonMessage(ctx, sbMsg, nil)
				return
			}
		} else {
			m.Body = sbMsg.Body
		}
	}

	p := &publication{
		topic:    sub.topic,
		msg:      &m,
		sbMsg:    sbMsg,
		receiver: receiver,
	}

	if err := handler(ctx, p); err != nil {
		p.err = err
		LogErrorf("handle message failed: %v", err)
		if eh := b.options.ErrorHandler; eh != nil {
			_ = eh(ctx, p)
		}
		_ = receiver.AbandonMessage(ctx, sbMsg, nil)
		return
	}

	if options.AutoAck {
		if err := p.Ack(); err != nil {
			LogErrorf("unable to ack msg: %v", err)
		}
	}
}

// isConflictError checks if the error is a 409 Conflict response
func isConflictError(err error) bool {
	var respErr *azcore.ResponseError
	if errors.As(err, &respErr) {
		return respErr.StatusCode == 409
	}
	// Fallback: check error message for conflict indicator
	return strings.Contains(err.Error(), "409") || strings.Contains(err.Error(), "Conflict")
}

// EnsureQueue ensures a queue exists with the given name.
func (b *azureBroker) EnsureQueue(ctx context.Context, queueName string, props *admin.QueueProperties) error {
	b.RLock()
	connStr := ""
	if b.options.Context != nil {
		if v, ok := b.options.Context.Value(connectionStringKey{}).(string); ok {
			connStr = v
		}
	}
	b.RUnlock()

	if connStr == "" {
		return errors.New("connection string is required")
	}

	adminClient, err := admin.NewClientFromConnectionString(connStr, nil)
	if err != nil {
		return fmt.Errorf("create admin client error: %w", err)
	}

	_, err = adminClient.CreateQueue(ctx, queueName, &admin.CreateQueueOptions{
		Properties: props,
	})
	if err != nil && !isConflictError(err) {
		return fmt.Errorf("create queue error: %w", err)
	}

	return nil
}

// EnsureTopic ensures a topic exists with the given name.
func (b *azureBroker) EnsureTopic(ctx context.Context, topicName string, props *admin.TopicProperties) error {
	b.RLock()
	connStr := ""
	if b.options.Context != nil {
		if v, ok := b.options.Context.Value(connectionStringKey{}).(string); ok {
			connStr = v
		}
	}
	b.RUnlock()

	if connStr == "" {
		return errors.New("connection string is required")
	}

	adminClient, err := admin.NewClientFromConnectionString(connStr, nil)
	if err != nil {
		return fmt.Errorf("create admin client error: %w", err)
	}

	_, err = adminClient.CreateTopic(ctx, topicName, &admin.CreateTopicOptions{
		Properties: props,
	})
	if err != nil && !isConflictError(err) {
		return fmt.Errorf("create topic error: %w", err)
	}

	return nil
}

// EnsureSubscription ensures a subscription exists for a topic.
func (b *azureBroker) EnsureSubscription(ctx context.Context, topicName, subscriptionName string, props *admin.SubscriptionProperties) error {
	b.RLock()
	connStr := ""
	if b.options.Context != nil {
		if v, ok := b.options.Context.Value(connectionStringKey{}).(string); ok {
			connStr = v
		}
	}
	b.RUnlock()

	if connStr == "" {
		return errors.New("connection string is required")
	}

	adminClient, err := admin.NewClientFromConnectionString(connStr, nil)
	if err != nil {
		return fmt.Errorf("create admin client error: %w", err)
	}

	_, err = adminClient.CreateSubscription(ctx, topicName, subscriptionName, &admin.CreateSubscriptionOptions{
		Properties: props,
	})
	if err != nil && !isConflictError(err) {
		return fmt.Errorf("create subscription error: %w", err)
	}

	return nil
}
