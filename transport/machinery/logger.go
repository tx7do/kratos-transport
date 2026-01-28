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
	level log.Level
}

func newLogger(level log.Level) logging.LoggerInterface {
	return &logger{
		level: level,
	}
}

func (l *logger) Print(args ...any) {
	LogInfo(args...)
}
func (l *logger) Printf(format string, args ...any) {
	LogInfof(format, args...)
}
func (l *logger) Println(args ...any) {
	LogInfo(args...)
}

func (l *logger) Fatal(args ...any) {
	LogFatal(args...)
}
func (l *logger) Fatalf(format string, args ...any) {
	LogFatalf(format, args...)
}
func (l *logger) Fatalln(args ...any) {
	LogFatal(args...)
}

func (l *logger) Panic(args ...any) {
	LogError(args...)
}
func (l *logger) Panicf(format string, args ...any) {
	LogErrorf(format, args...)
}
func (l *logger) Panicln(args ...any) {
	LogError(args...)
}
