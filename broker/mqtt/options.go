package mqtt

import (
	"github.com/tx7do/kratos-transport/broker"
)

///
/// Option
///

type cleanSessionKey struct{}
type authKey struct{}
type clientIdKey struct{}
type autoReconnectKey struct{}
type resumeSubsKey struct{}
type orderMattersKey struct{}

type AuthRecord struct {
	Username string
	Password string
}

// WithCleanSession enable clean session option
func WithCleanSession(enable bool) broker.Option {
	return broker.OptionContextWithValue(cleanSessionKey{}, enable)
}

// WithAuth set username & password options
func WithAuth(username string, password string) broker.Option {
	return broker.OptionContextWithValue(authKey{}, &AuthRecord{
		Username: username,
		Password: password,
	})
}

// WithClientId set client id option
func WithClientId(clientId string) broker.Option {
	return broker.OptionContextWithValue(clientIdKey{}, clientId)
}

// WithAutoReconnect enable aut reconnect option
func WithAutoReconnect(enable bool) broker.Option {
	return broker.OptionContextWithValue(autoReconnectKey{}, enable)
}

// WithResumeSubs .
func WithResumeSubs(enable bool) broker.Option {
	return broker.OptionContextWithValue(resumeSubsKey{}, enable)
}

// WithOrderMatters .
func WithOrderMatters(enable bool) broker.Option {
	return broker.OptionContextWithValue(orderMattersKey{}, enable)
}

///
/// SubscribeOption
///

type qosSubscribeKey struct{}

// WithSubscribeQos QOS
func WithSubscribeQos(qos byte) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(qosSubscribeKey{}, qos)
}

///
/// PublishOption
///

type qosPublishKey struct{}
type retainedPublishKey struct{}

// WithPublishQos QOS
func WithPublishQos(qos byte) broker.PublishOption {
	return broker.PublishContextWithValue(qosPublishKey{}, qos)
}

// WithPublishRetained retained
func WithPublishRetained(qos byte) broker.PublishOption {
	return broker.PublishContextWithValue(retainedPublishKey{}, qos)
}
