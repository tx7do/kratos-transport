package nats

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/otel/trace"

	natsGo "github.com/nats-io/nats.go"

	"github.com/tx7do/kratos-transport/broker"
)

///////////////////////////////////////////////////////////////////////////////
/// JetStream Publish
///////////////////////////////////////////////////////////////////////////////

func (b *jetStreamBroker) Publish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	var finalTask = b.internalPublish

	if len(b.options.PublishMiddlewares) > 0 {
		finalTask = broker.ChainPublishMiddleware(finalTask, b.options.PublishMiddlewares)
	}

	return finalTask(ctx, topic, msg, opts...)
}

func (b *jetStreamBroker) internalPublish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	buf, err := broker.Marshal(b.options.Codec, msg.Body)
	if err != nil {
		return err
	}

	sendMsg := msg.Clone()
	sendMsg.Body = buf

	return b.publish(ctx, topic, sendMsg, opts...)
}

func (b *jetStreamBroker) publish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	b.RLock()
	defer b.RUnlock()

	if b.js == nil {
		return errors.New("not connected")
	}

	publishOpts := broker.NewPublishOptions(opts...)

	m := natsGo.NewMsg(topic)
	m.Data = msg.BodyBytes()

	// Apply headers
	if headers, ok := publishOpts.Context.Value(headersKey{}).(map[string][]string); ok {
		for k, v := range headers {
			for _, vv := range v {
				m.Header.Add(k, vv)
			}
		}
	}

	// Build JetStream publish options
	var pubOpts []natsGo.PubOpt

	if v, ok := publishOpts.Context.Value(pubMsgIdKey{}).(string); ok && v != "" {
		pubOpts = append(pubOpts, natsGo.MsgId(v))
	}
	if v, ok := publishOpts.Context.Value(pubExpectStreamKey{}).(string); ok && v != "" {
		pubOpts = append(pubOpts, natsGo.ExpectStream(v))
	}
	if v, ok := publishOpts.Context.Value(pubExpectLastSequenceKey{}).(uint64); ok {
		pubOpts = append(pubOpts, natsGo.ExpectLastSequence(v))
	}
	if v, ok := publishOpts.Context.Value(pubExpectLastSequencePerSubjectKey{}).(uint64); ok {
		pubOpts = append(pubOpts, natsGo.ExpectLastSequencePerSubject(v))
	}
	if v, ok := publishOpts.Context.Value(pubExpectLastMsgIdKey{}).(string); ok && v != "" {
		pubOpts = append(pubOpts, natsGo.ExpectLastMsgId(v))
	}
	if v, ok := publishOpts.Context.Value(pubRawOptsKey{}).([]natsGo.PubOpt); ok {
		pubOpts = append(pubOpts, v...)
	}

	var span trace.Span
	ctx, span = b.startProducerSpan(publishOpts.Context, m)

	_, err := b.js.PublishMsg(m, pubOpts...)

	b.finishProducerSpan(ctx, span, err)

	return err
}

///////////////////////////////////////////////////////////////////////////////
/// JetStream Request (not supported in JetStream — falls back to core NATS Request)
///////////////////////////////////////////////////////////////////////////////

func (b *jetStreamBroker) Request(ctx context.Context, topic string, msg *broker.Message, opts ...broker.RequestOption) (*broker.Message, error) {
	if msg == nil {
		return nil, broker.ErrRequestMessageNil
	}
	b.RLock()
	defer b.RUnlock()

	if b.conn == nil {
		return nil, errors.New("not connected")
	}

	options := broker.NewRequestOptions(opts...)

	buf, err := broker.Marshal(b.options.Codec, msg.Body)
	if err != nil {
		return nil, err
	}

	m := natsGo.NewMsg(topic)
	m.Data = buf

	// 标准 RequestOptions.Timeout 优先（WithRequestTimeout），专有 key 作为兜底
	var timeout = time.Second * 2
	if options.Timeout > 0 {
		timeout = options.Timeout
	}
	if v, ok := options.Context.Value(requestTimeoutKey{}).(time.Duration); ok && v > 0 {
		timeout = v
	}

	if headers, ok := options.Context.Value(headersKey{}).(map[string][]string); ok {
		for k, v := range headers {
			for _, vv := range v {
				m.Header.Add(k, vv)
			}
		}
	}

	var span trace.Span
	ctx, span = b.startProducerSpan(options.Context, m)

	res, err := b.conn.RequestMsg(m, timeout)

	b.finishProducerSpan(ctx, span, err)

	if err != nil {
		return nil, err
	}

	return broker.NewMessage(res, broker.WithMsg(res)), nil
}
