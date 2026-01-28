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

func LogDebug(args ...any) {
	log.Debugf("%s %s", logKey, fmt.Sprint(args...))
}

func LogInfo(args ...any) {
	log.Infof("%s %s", logKey, fmt.Sprint(args...))
}

func LogWarn(args ...any) {
	log.Warnf("%s %s", logKey, fmt.Sprint(args...))
}

func LogError(args ...any) {
	log.Errorf("%s %s", logKey, fmt.Sprint(args...))
}

func LogFatal(args ...any) {
	log.Fatalf("%s %s", logKey, fmt.Sprint(args...))
}

///
/// logger
///

func LogDebugf(format string, args ...any) {
	log.Debugf("%s %s", logKey, fmt.Sprintf(format, args...))
}

func LogInfof(format string, args ...any) {
	log.Infof("%s %s", logKey, fmt.Sprintf(format, args...))
}

func LogWarnf(format string, args ...any) {
	log.Warnf("%s %s", logKey, fmt.Sprintf(format, args...))
}

func LogErrorf(format string, args ...any) {
	log.Errorf("%s %s", logKey, fmt.Sprintf(format, args...))
}

func LogFatalf(format string, args ...any) {
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

func (l logger) Debug(args ...any) {
	LogDebug(args...)
}

func (l logger) Info(args ...any) {
	LogInfo(args...)
}

func (l logger) Warn(args ...any) {
	LogWarn(args...)
}

func (l logger) Error(args ...any) {
	LogError(args...)
}

func (l logger) Fatal(args ...any) {
	LogFatal(args...)
}
