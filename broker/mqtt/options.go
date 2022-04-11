package mqtt

import (
	"context"
	"github.com/tx7do/kratos-transport/broker"
)

type cleanSessionKey struct{}
type authKey struct{}
type clientIdKey struct{}

type authRecord struct {
	username string
	password string
}

func WithCleanSession(enable bool) broker.Option {
	return broker.OptionContextWithValue(cleanSessionKey{}, enable)
}

func CleanSessionFromContext(ctx context.Context) (bool, bool) {
	v, ok := ctx.Value(cleanSessionKey{}).(bool)
	return v, ok
}

func WithAuth(username string, password string) broker.Option {
	return broker.OptionContextWithValue(authKey{}, &authRecord{
		username: username,
		password: password,
	})
}

func AuthFromContext(ctx context.Context) (*authRecord, bool) {
	v, ok := ctx.Value(authKey{}).(*authRecord)
	return v, ok
}

func WithClientId(clientId string) broker.Option {
	return broker.OptionContextWithValue(clientIdKey{}, clientId)
}

func ClientIdFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(clientIdKey{}).(string)
	return v, ok
}
