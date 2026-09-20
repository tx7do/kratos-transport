package goworkflows

import (
	"fmt"
	"os"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
)

const (
	logKey = "[GoWorkflows]"
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
func logAt(level log.Level, msg string) {
	msg = logKey + " " + msg
	if l := getLogger(); l != nil {
		_ = l.Log(level, log.DefaultMessageKey, msg)
		if level == log.LevelFatal {
			os.Exit(1)
		}
		return
	}
	log.Log(level, log.DefaultMessageKey, msg)
	if level == log.LevelFatal {
		os.Exit(1)
	}
}

// logAtf 为 logAt 的格式化版本
func logAtf(level log.Level, format string, args ...any) {
	logAt(level, fmt.Sprintf(format, args...))
}

func LogDebug(args ...any) {
	logAt(log.LevelDebug, fmt.Sprint(args...))
}

func LogInfo(args ...any) {
	logAt(log.LevelInfo, fmt.Sprint(args...))
}

func LogWarn(args ...any) {
	logAt(log.LevelWarn, fmt.Sprint(args...))
}

func LogError(args ...any) {
	logAt(log.LevelError, fmt.Sprint(args...))
}

func LogFatal(args ...any) {
	logAt(log.LevelFatal, fmt.Sprint(args...))
}

func LogDebugf(format string, args ...any) {
	logAtf(log.LevelDebug, format, args...)
}

func LogInfof(format string, args ...any) {
	logAtf(log.LevelInfo, format, args...)
}

func LogWarnf(format string, args ...any) {
	logAtf(log.LevelWarn, format, args...)
}

func LogErrorf(format string, args ...any) {
	logAtf(log.LevelError, format, args...)
}

func LogFatalf(format string, args ...any) {
	logAtf(log.LevelFatal, format, args...)
}
