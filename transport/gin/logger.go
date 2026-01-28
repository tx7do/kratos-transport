package gin

import (
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
)

const (
	logKey = "[" + KindGin + "]"
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

type infoLogger struct {
	Logger log.Logger
}

func (l *infoLogger) Write(p []byte) (n int, err error) {
	err = l.Logger.Log(log.LevelInfo, "msg", string(p))
	return
}

type errLogger struct {
	Logger log.Logger
}

func (l *errLogger) Write(p []byte) (n int, err error) {
	err = l.Logger.Log(log.LevelError, "msg", string(p))
	return
}
