package gin

import "github.com/go-kratos/kratos/v2/log"

type infoLogger struct {
	Logger log.Logger
}

func (l *infoLogger) Write(p []byte) (n int, err error) {
	err = l.Logger.Log(log.LevelInfo, "info", string(p))
	return
}

type errLogger struct {
	Logger log.Logger
}

func (l *errLogger) Write(p []byte) (n int, err error) {
	err = l.Logger.Log(log.LevelError, "error", string(p))
	return
}
