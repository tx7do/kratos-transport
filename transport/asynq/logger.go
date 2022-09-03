package asynq

import (
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/hibiken/asynq"
)

const (
	logKey = "asynq"
)

type logger struct {
}

func newLogger() asynq.Logger {
	return &logger{}
}

func (l *logger) Debug(args ...interface{}) {
	_ = log.GetLogger().Log(log.LevelDebug, logKey, fmt.Sprint(args...))
}

func (l *logger) Info(args ...interface{}) {
	_ = log.GetLogger().Log(log.LevelInfo, logKey, fmt.Sprint(args...))
}

func (l *logger) Warn(args ...interface{}) {
	_ = log.GetLogger().Log(log.LevelWarn, logKey, fmt.Sprint(args...))
}

func (l *logger) Error(args ...interface{}) {
	_ = log.GetLogger().Log(log.LevelError, logKey, fmt.Sprint(args...))
}

func (l *logger) Fatal(args ...interface{}) {
	_ = log.GetLogger().Log(log.LevelFatal, logKey, fmt.Sprint(args...))
}
