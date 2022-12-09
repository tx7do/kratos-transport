package mqtt

import (
	"github.com/tx7do/kratos-transport/broker"
)

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

func WithCleanSession(enable bool) broker.Option {
	return broker.OptionContextWithValue(cleanSessionKey{}, enable)
}

func WithAuth(username string, password string) broker.Option {
	return broker.OptionContextWithValue(authKey{}, &AuthRecord{
		Username: username,
		Password: password,
	})
}

func WithClientId(clientId string) broker.Option {
	return broker.OptionContextWithValue(clientIdKey{}, clientId)
}

func WithAutoReconnect(enable bool) broker.Option {
	return broker.OptionContextWithValue(autoReconnectKey{}, enable)
}

func WithResumeSubs(enable bool) broker.Option {
	return broker.OptionContextWithValue(resumeSubsKey{}, enable)
}

func WithOrderMatters(enable bool) broker.Option {
	return broker.OptionContextWithValue(orderMattersKey{}, enable)
}
