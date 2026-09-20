package keepalive

import (
	"fmt"
	"os"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
)

var (
	libLoggerMu sync.RWMutex
	libLogger   log.Logger
)

// SetLogger 注入库内部日志使用的 logger，传入 nil 恢复为 kratos 全局 logger。
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

func logAt(level log.Level, msg string) {
	if l := getLogger(); l != nil {
		l.Log(level, log.DefaultMessageKey, msg)
	} else {
		log.Log(level, log.DefaultMessageKey, msg)
	}
	if level == log.LevelFatal {
		os.Exit(1)
	}
}

func logInfof(format string, args ...interface{}) {
	logAt(log.LevelInfo, fmt.Sprintf(format, args...))
}

func logWarnf(format string, args ...interface{}) {
	logAt(log.LevelWarn, fmt.Sprintf(format, args...))
}
