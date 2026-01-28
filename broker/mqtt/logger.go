package mqtt

import (
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
)

const (
	logKey = "[mqtt]"
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
/// ErrorLogger
///

type ErrorLogger struct{}

func (ErrorLogger) Println(v ...any) {
	log.Error(v...)
}

func (ErrorLogger) Printf(format string, v ...any) {
	log.Errorf(format, v...)
}

///
/// CriticalLogger
///

type CriticalLogger struct{}

func (CriticalLogger) Println(v ...any) {
	log.Fatal(v...)
}

func (CriticalLogger) Printf(format string, v ...any) {
	log.Fatalf(format, v...)
}

///
/// WarnLogger
///

type WarnLogger struct{}

func (WarnLogger) Println(v ...any) {
	log.Warn(v...)
}

func (WarnLogger) Printf(format string, v ...any) {
	log.Warnf(format, v...)
}

///
/// DebugLogger
///

type DebugLogger struct{}

func (DebugLogger) Println(v ...any) {
	log.Debug(v...)
}

func (DebugLogger) Printf(format string, v ...any) {
	log.Debugf(format, v...)
}
