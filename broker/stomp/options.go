package stomp

import (
	"context"
	"time"

	"github.com/tx7do/kratos-transport/broker"
)

///////////////////////////////////////////////////////////////////////////////

type authKey struct{}
type connectTimeoutKey struct{}
type connectHeaderKey struct{}
type vHostKey struct{}
type heartBeatKey struct{}
type heartBeatErrorKey struct{}
type msgSendTimeoutKey struct{}
type rcvReceiptTimeoutKey struct{}

type authRecord struct {
	username string
	password string
}
type heartbeatTimeout struct {
	sendTimeout time.Duration
	recvTimeout time.Duration
}

func WithConnectTimeout(ct time.Duration) broker.Option {
	return broker.OptionContextWithValue(connectTimeoutKey{}, ct)
}

func WithAuth(username string, password string) broker.Option {
	return broker.OptionContextWithValue(authKey{}, &authRecord{
		username: username,
		password: password,
	})
}

func WithConnectHeaders(h map[string]string) broker.Option {
	return broker.OptionContextWithValue(connectHeaderKey{}, h)
}

func WithVirtualHost(h string) broker.Option {
	return broker.OptionContextWithValue(vHostKey{}, h)
}

func WithHeartBeat(sendTimeout, recvTimeout time.Duration) broker.Option {
	d := &heartbeatTimeout{
		sendTimeout: sendTimeout,
		recvTimeout: recvTimeout,
	}
	return broker.OptionContextWithValue(heartBeatKey{}, d)
}

func WithHeartBeatError(errorTimeout time.Duration) broker.Option {
	return broker.OptionContextWithValue(heartBeatErrorKey{}, errorTimeout)
}

func WithMsgSendTimeout(tm time.Duration) broker.Option {
	return broker.OptionContextWithValue(msgSendTimeoutKey{}, tm)
}

func WithRcvReceiptTimeout(tm time.Duration) broker.Option {
	return broker.OptionContextWithValue(rcvReceiptTimeoutKey{}, tm)
}

///////////////////////////////////////////////////////////////////////////////

type receiptKey struct{}
type headerKey struct{}
type suppressContentLengthKey struct{}

func WithReceipt(_ time.Duration) broker.PublishOption {
	return broker.PublishContextWithValue(receiptKey{}, true)
}

func WithSuppressContentLength(_ time.Duration) broker.PublishOption {
	return broker.PublishContextWithValue(suppressContentLengthKey{}, true)
}

func WithHeaders(h map[string]string) broker.PublishOption {
	return broker.PublishContextWithValue(headerKey{}, h)
}

///////////////////////////////////////////////////////////////////////////////

type durableQueueKey struct{}
type subscribeHeaderKey struct{}
type subscribeContextKey struct{}
type ackSuccessKey struct{}

func WithSubscribeHeaders(headers map[string]string) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(subscribeHeaderKey{}, headers)
}

func WithSubscribeContext(ctx context.Context) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(subscribeContextKey{}, ctx)
}

func WithAckOnSuccess() broker.SubscribeOption {
	return broker.SubscribeContextWithValue(ackSuccessKey{}, true)
}

func WithDurable() broker.SubscribeOption {
	return broker.SubscribeContextWithValue(durableQueueKey{}, true)
}
