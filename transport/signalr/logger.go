package signalr

import "github.com/go-kratos/kratos/v2/log"

type logger struct {
	log.Logger
}

func (l *logger) Log(keyVals ...interface{}) error {
	return l.Logger.Log(log.LevelError, keyVals...)
}
