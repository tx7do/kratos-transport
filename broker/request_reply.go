package broker

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// request-reply 相关的标准消息头。
// 请求方通过 [GenericRequest] 自动携带；应答方通过 [ReplyTo] 读取并回应。
const (
	// HeaderReplyTo 响应应投递到的主题
	HeaderReplyTo = "Reply-To"
	// HeaderCorrelationID 用于在共享主题上关联请求与响应
	HeaderCorrelationID = "Correlation-ID"
)

var (
	// ErrRequestMessageNil 请求消息为空
	ErrRequestMessageNil = errors.New("request message is nil")
	// ErrRequestNoReplyTo 请求消息缺少 Reply-To 头，无法回应
	ErrRequestNoReplyTo = errors.New("request message has no Reply-To header")
)

// GenericRequest 是 Request 的通用实现：
//
// 1. 订阅一个（自动生成的或由 WithReplyTopic 指定的）回复主题；
// 2. 向 topic 发布请求，自动附带 Reply-To 与 Correlation-ID 头；
// 3. 阻塞等待 Correlation-ID 匹配的响应，直到超时或 ctx 取消。
//
// 它只依赖 Broker 接口自身的 Publish/Subscribe 语义，适用于所有消息型驱动；
// 拥有原生 request-reply 能力的驱动（如 nats）应优先使用原生实现。
//
// 注意：应答方必须把响应发布到请求的 Reply-To 主题并回传 Correlation-ID，
// 可直接使用 [ReplyTo] 辅助函数。
func GenericRequest(ctx context.Context, b Broker, topic string, msg *Message, opts ...RequestOption) (*Message, error) {
	if b == nil {
		return nil, errors.New("broker is nil")
	}
	if msg == nil {
		return nil, ErrRequestMessageNil
	}

	options := NewRequestOptions(opts...)
	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}

	replyTopic := options.ReplyTopic
	if replyTopic == "" {
		replyTopic = "kratos.reply." + uuid.NewString()
	}
	if msg.ID == "" {
		msg.ID = uuid.NewString()
	}

	if msg.Headers == nil {
		msg.Headers = make(Headers)
	}
	msg.Headers[HeaderReplyTo] = replyTopic
	msg.Headers[HeaderCorrelationID] = msg.ID

	// 先订阅回复主题，再发布请求，避免响应早于订阅到达而丢失
	replyCh := make(chan *Message, 1)
	handler := func(ctx context.Context, event Event) error {
		if event == nil || event.Message() == nil {
			return nil
		}
		m := event.Message()
		// 只接受与本请求关联的响应
		if m.GetHeader(HeaderCorrelationID) != msg.ID {
			return nil
		}
		select {
		case replyCh <- m:
		default:
		}
		return nil
	}

	sub, err := b.Subscribe(replyTopic, handler, nil, WithSubscribeAutoAck(true))
	if err != nil {
		return nil, fmt.Errorf("subscribe reply topic %s failed: %w", replyTopic, err)
	}
	defer func() {
		_ = sub.Unsubscribe(false)
	}()

	if err = b.Publish(ctx, topic, msg); err != nil {
		return nil, fmt.Errorf("publish request failed: %w", err)
	}

	select {
	case reply := <-replyCh:
		return reply, nil
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("request timeout after %s", options.Timeout)
		}
		return nil, ctx.Err()
	}
}

// NewReply 基于请求构造响应消息：自动回传 Correlation-ID。
func NewReply(request *Message, body any) *Message {
	reply := NewMessage(body)
	if request != nil {
		if cid := request.GetHeader(HeaderCorrelationID); cid != "" {
			reply.SetHeader(HeaderCorrelationID, cid)
		}
	}
	return reply
}

// ReplyTo 把响应发布到请求的 Reply-To 主题，供请求方 [GenericRequest] 接收。
func ReplyTo(ctx context.Context, b Broker, request *Message, body any, opts ...PublishOption) error {
	if b == nil {
		return errors.New("broker is nil")
	}
	if request == nil {
		return ErrRequestMessageNil
	}

	replyTopic := request.GetHeader(HeaderReplyTo)
	if replyTopic == "" {
		return ErrRequestNoReplyTo
	}

	return b.Publish(ctx, replyTopic, NewReply(request, body), opts...)
}
