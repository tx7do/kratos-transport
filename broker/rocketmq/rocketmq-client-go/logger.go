package rocketmqClientGo

import (
	"fmt"
	"strings"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
)

const (
	loggerKey = "[rocketmq] "
)

var (
	libLoggerMu sync.RWMutex
	libLogger   log.Logger
)

// SetLogger 注入库内部日志使用的 logger，传入 nil 恢复为 kratos 全局 logger。
// 默认使用 kratos 全局 logger（无级别过滤），也可通过 broker.WithLogger 选项在构造时注入。
// 注意：logger 为包级生效，后创建的 broker 会覆盖先注入的。
func SetLogger(l log.Logger) {
	libLoggerMu.Lock()
	defer libLoggerMu.Unlock()
	libLogger = l
}

func getLogger() log.Logger {
	libLoggerMu.RLock()
	defer libLoggerMu.RUnlock()
	return libLogger
}

// logAt 经由注入的 logger 输出日志，未注入时退回 kratos 全局 logger
func logAt(level log.Level, keyVals ...any) {
	if l := getLogger(); l != nil {
		_ = l.Log(level, keyVals...)
		return
	}
	log.Log(level, keyVals...)
}

type logger struct {
	level log.Level
}

func toKeyVals(fields map[string]any) (keyVals []any) {
	for k, v := range fields {
		keyVals = append(keyVals, k)
		keyVals = append(keyVals, v)
	}
	return
}

func (l *logger) Log(level log.Level, msg string, fields map[string]any) {
	if l.level > level {
		return
	}

	keyVals := toKeyVals(fields)
	logAt(level, loggerKey+msg, keyVals)
}

func (l *logger) Logf(level log.Level, format string, a ...any) {
	if l.level > level {
		return
	}
	var keyVals []any
	keyVals = append(keyVals, loggerKey)
	logAt(level, fmt.Sprintf(format, a...))
}

func (l *logger) Debug(msg string, fields map[string]any) {
	l.Log(log.LevelDebug, msg, fields)
}

func (l *logger) Debugf(format string, a ...any) {
	l.Logf(log.LevelDebug, format, a...)
}

func (l *logger) Info(msg string, fields map[string]any) {
	l.Log(log.LevelInfo, msg, fields)
}

func (l *logger) Infof(format string, a ...any) {
	l.Logf(log.LevelInfo, format, a...)
}

func (l *logger) Warning(msg string, fields map[string]any) {
	l.Log(log.LevelWarn, msg, fields)
}

func (l *logger) Warningf(format string, a ...any) {
	l.Logf(log.LevelWarn, format, a...)
}

func (l *logger) Error(msg string, fields map[string]any) {
	l.Log(log.LevelError, msg, fields)
}

func (l *logger) Errorf(format string, a ...any) {
	l.Logf(log.LevelError, format, a...)
}

func (l *logger) Fatal(msg string, fields map[string]any) {
	l.Log(log.LevelFatal, msg, fields)
}

func (l *logger) Fatalf(format string, a ...any) {
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
