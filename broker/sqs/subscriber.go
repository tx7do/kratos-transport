package sqs

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"github.com/tx7do/kratos-transport/broker"
)

type subscriber struct {
	sync.RWMutex

	topic    string
	queueUrl string
	options  broker.SubscribeOptions

	b      *sqsBroker
	client *sqs.Client
	cancel context.CancelFunc
	closed bool
}

func (s *subscriber) Options() broker.SubscribeOptions {
	s.RLock()
	defer s.RUnlock()
	return s.options
}

func (s *subscriber) Topic() string {
	s.RLock()
	defer s.RUnlock()
	return s.topic
}

func (s *subscriber) Unsubscribe(removeFromManager bool) error {
	s.Lock()
	defer s.Unlock()

	if s.cancel != nil {
		s.cancel()
	}

	s.closed = true

	if s.b != nil && s.b.subscribers != nil && removeFromManager {
		_ = s.b.subscribers.RemoveOnly(s.topic)
	}

	return nil
}

func (s *subscriber) IsClosed() bool {
	s.RLock()
	defer s.RUnlock()
	return s.closed
}

// recv is the main receive loop for SQS messages using long polling.
func (s *subscriber) recv(ctx context.Context, handler broker.Handler, binder broker.Binder, opts recvOpts) {
	defer func() {
		LogInfof("subscriber stopped, topic: %s", s.topic)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		result, err := s.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            &s.queueUrl,
			MaxNumberOfMessages: opts.maxMessages,
			WaitTimeSeconds:     opts.waitTimeSeconds,
			VisibilityTimeout:   opts.visibilityTimeout,
			AttributeNames: []types.QueueAttributeName{
				types.QueueAttributeNameAll,
			},
			MessageAttributeNames: []string{
				"All",
			},
		})
		if err != nil {
			// context canceled is expected during shutdown
			if ctx.Err() != nil {
				return
			}
			LogErrorf("receive message failed: %v", err)
			continue
		}

		if len(result.Messages) == 0 {
			continue
		}

		for _, sqsMsg := range result.Messages {
			s.processMessage(ctx, handler, binder, sqsMsg)
		}
	}
}

func (s *subscriber) processMessage(ctx context.Context, handler broker.Handler, binder broker.Binder, sqsMsg types.Message) {
	var m broker.Message

	// Extract headers from message attributes
	if sqsMsg.MessageAttributes != nil {
		m.Headers = make(broker.Headers)
		for k, v := range sqsMsg.MessageAttributes {
			if v.StringValue != nil {
				m.Headers[k] = *v.StringValue
			}
		}
	}

	// Extract body
	if sqsMsg.Body != nil {
		body := []byte(*sqsMsg.Body)

		if binder != nil {
			m.Body = binder()
			if err := broker.Unmarshal(s.b.options.Codec, body, &m.Body); err != nil {
				LogErrorf("unmarshal message failed: %v", err)
				return
			}
		} else {
			m.Body = body
		}
	}

	p := &publication{
		topic:    s.topic,
		msg:      &m,
		sqsMsg:   &sqsMsg,
		client:   s.client,
		queueUrl: s.queueUrl,
	}

	if err := handler(ctx, p); err != nil {
		p.err = err
		LogErrorf("handle message failed: %v", err)
		if eh := s.b.options.ErrorHandler; eh != nil {
			_ = eh(ctx, p)
		}
		return
	}

	if s.options.AutoAck {
		if err := p.Ack(); err != nil {
			LogErrorf("unable to ack msg: %v", err)
		}
	}
}

// recvOpts holds the SQS receive parameters.
type recvOpts struct {
	visibilityTimeout int32
	waitTimeSeconds   int32
	maxMessages       int32
}
