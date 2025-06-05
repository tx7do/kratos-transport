package asynq

import (
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/hibiken/asynq"
)

const (
	logKey = "[" + KindAsynq + "]"
)

///
/// logger
///

func LogDebug(args ...interface{}) {
	log.Debugf("%s %s", logKey, fmt.Sprint(args...))
}

func LogInfo(args ...interface{}) {
	log.Infof("%s %s", logKey, fmt.Sprint(args...))
}

func LogWarn(args ...interface{}) {
	log.Warnf("%s %s", logKey, fmt.Sprint(args...))
}

func LogError(args ...interface{}) {
	log.Errorf("%s %s", logKey, fmt.Sprint(args...))
}

func LogFatal(args ...interface{}) {
	log.Fatalf("%s %s", logKey, fmt.Sprint(args...))
}

///
/// logger
///

func LogDebugf(format string, args ...interface{}) {
	log.Debugf("%s %s", logKey, fmt.Sprintf(format, args...))
}

func LogInfof(format string, args ...interface{}) {
	log.Infof("%s %s", logKey, fmt.Sprintf(format, args...))
}

func LogWarnf(format string, args ...interface{}) {
	log.Warnf("%s %s", logKey, fmt.Sprintf(format, args...))
}

func LogErrorf(format string, args ...interface{}) {
	log.Errorf("%s %s", logKey, fmt.Sprintf(format, args...))
}

func LogFatalf(format string, args ...interface{}) {
	log.Fatalf("%s %s", logKey, fmt.Sprintf(format, args...))
}

///
/// logger
///

type logger struct {
}

func newLogger() asynq.Logger {
	return &logger{}
}

func (l logger) Debug(args ...interface{}) {
	LogDebug(args...)
}

func (l logger) Info(args ...interface{}) {
	LogInfo(args...)
}

func (l logger) Warn(args ...interface{}) {
	LogWarn(args...)
}

func (l logger) Error(args ...interface{}) {
	LogError(args...)
}

func (l logger) Fatal(args ...interface{}) {
	LogFatal(args...)
}
