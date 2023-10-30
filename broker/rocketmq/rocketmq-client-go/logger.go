package rocketmqClientGo

import (
	"fmt"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
)

const (
	loggerKey = "[rocketmq] "
)

type logger struct {
	level log.Level
}

func toKeyVals(fields map[string]interface{}) (keyVals []interface{}) {
	for k, v := range fields {
		keyVals = append(keyVals, k)
		keyVals = append(keyVals, v)
	}
	return
}

func (l *logger) Log(level log.Level, msg string, fields map[string]interface{}) {
	if l.level > level {
		return
	}

	keyVals := toKeyVals(fields)
	log.Log(level, loggerKey+msg, keyVals)
}

func (l *logger) Logf(level log.Level, format string, a ...interface{}) {
	if l.level > level {
		return
	}
	var keyVals []interface{}
	keyVals = append(keyVals, loggerKey)
	log.Log(level, fmt.Sprintf(format, a...))
}

func (l *logger) Debug(msg string, fields map[string]interface{}) {
	l.Log(log.LevelDebug, msg, fields)
}

func (l *logger) Debugf(format string, a ...interface{}) {
	l.Logf(log.LevelDebug, format, a...)
}

func (l *logger) Info(msg string, fields map[string]interface{}) {
	l.Log(log.LevelInfo, msg, fields)
}

func (l *logger) Infof(format string, a ...interface{}) {
	l.Logf(log.LevelInfo, format, a...)
}

func (l *logger) Warning(msg string, fields map[string]interface{}) {
	l.Log(log.LevelWarn, msg, fields)
}

func (l *logger) Warningf(format string, a ...interface{}) {
	l.Logf(log.LevelInfo, format, a...)
}

func (l *logger) Error(msg string, fields map[string]interface{}) {
	l.Log(log.LevelError, msg, fields)
}

func (l *logger) Errorf(format string, a ...interface{}) {
	l.Logf(log.LevelError, format, a...)
}

func (l *logger) Fatal(msg string, fields map[string]interface{}) {
	l.Log(log.LevelFatal, msg, fields)
}

func (l *logger) Fatalf(format string, a ...interface{}) {
	l.Logf(log.LevelFatal, format, a...)
}

func (l *logger) Level(lvl string) {
	switch strings.ToLower(lvl) {
	case "panic":
		l.level = log.LevelFatal
	case "fatal":
		l.level = log.LevelFatal
	case "error":
		l.level = log.LevelError
	case "warn", "warning":
		l.level = log.LevelWarn
	case "info":
		l.level = log.LevelInfo
	case "debug":
		l.level = log.LevelDebug
	case "trace":
		l.level = log.LevelDebug
	}
}

func (l *logger) OutputPath(_ string) (err error) {
	return nil
}
