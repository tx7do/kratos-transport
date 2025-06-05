package machinery

import (
	"fmt"

	"github.com/RichardKnop/logging"
	"github.com/go-kratos/kratos/v2/log"
)

const (
	logKey = "[" + KindMachinery + "]"
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
	level log.Level
}

func newLogger(level log.Level) logging.LoggerInterface {
	return &logger{
		level: level,
	}
}

func (l *logger) Print(args ...interface{}) {
	LogInfo(args...)
}
func (l *logger) Printf(format string, args ...interface{}) {
	LogInfof(format, args...)
}
func (l *logger) Println(args ...interface{}) {
	LogInfo(args...)
}

func (l *logger) Fatal(args ...interface{}) {
	LogFatal(args...)
}
func (l *logger) Fatalf(format string, args ...interface{}) {
	LogFatalf(format, args...)
}
func (l *logger) Fatalln(args ...interface{}) {
	LogFatal(args...)
}

func (l *logger) Panic(args ...interface{}) {
	LogError(args...)
}
func (l *logger) Panicf(format string, args ...interface{}) {
	LogErrorf(format, args...)
}
func (l *logger) Panicln(args ...interface{}) {
	LogError(args...)
}
