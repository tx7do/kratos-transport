package mqtt

import (
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
)

const (
	logKey = "[" + KindMQTT + "]"
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
