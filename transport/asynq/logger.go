package asynq

import (
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/hibiken/asynq"
)

const (
	logKey = "asynq"
)

///
/// logger
///

func LogDebug(args ...interface{}) {
	_ = log.GetLogger().Log(log.LevelDebug, logKey, fmt.Sprint(args...))
}

func LogInfo(args ...interface{}) {
	_ = log.GetLogger().Log(log.LevelInfo, logKey, fmt.Sprint(args...))
}

func LogWarn(args ...interface{}) {
	_ = log.GetLogger().Log(log.LevelWarn, logKey, fmt.Sprint(args...))
}

func LogError(args ...interface{}) {
	_ = log.GetLogger().Log(log.LevelError, logKey, fmt.Sprint(args...))
}

func LogFatal(args ...interface{}) {
	_ = log.GetLogger().Log(log.LevelFatal, logKey, fmt.Sprint(args...))
}

///
/// logger
///

func LogDebugf(format string, args ...interface{}) {
	_ = log.GetLogger().Log(log.LevelDebug, logKey, fmt.Sprintf(format, args...))
}

func LogInfof(format string, args ...interface{}) {
	_ = log.GetLogger().Log(log.LevelInfo, logKey, fmt.Sprintf(format, args...))
}

func LogWarnf(format string, args ...interface{}) {
	_ = log.GetLogger().Log(log.LevelWarn, logKey, fmt.Sprintf(format, args...))
}

func LogErrorf(format string, args ...interface{}) {
	_ = log.GetLogger().Log(log.LevelError, logKey, fmt.Sprintf(format, args...))
}

func LogFatalf(format string, args ...interface{}) {
	_ = log.GetLogger().Log(log.LevelFatal, logKey, fmt.Sprintf(format, args...))
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
