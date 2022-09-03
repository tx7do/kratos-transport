package asynq

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/hibiken/asynq"
)

type logger struct {
}

func newLogger() asynq.Logger {
	return &logger{}
}

func (l *logger) Debug(args ...interface{}) {
	log.Debug(args...)
}

func (l *logger) Info(args ...interface{}) {
	log.Info(args...)
}

func (l *logger) Warn(args ...interface{}) {
	log.Warn(args...)
}

func (l *logger) Error(args ...interface{}) {
	log.Error(args...)
}

func (l *logger) Fatal(args ...interface{}) {
	log.Fatal(args...)
}
