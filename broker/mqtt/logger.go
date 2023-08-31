package mqtt

import "github.com/go-kratos/kratos/v2/log"

///
/// ErrorLogger
///

type ErrorLogger struct{}

func (ErrorLogger) Println(v ...interface{}) {
	log.Error(v...)
}

func (ErrorLogger) Printf(format string, v ...interface{}) {
	log.Errorf(format, v...)
}

///
/// CriticalLogger
///

type CriticalLogger struct{}

func (CriticalLogger) Println(v ...interface{}) {
	log.Fatal(v...)
}

func (CriticalLogger) Printf(format string, v ...interface{}) {
	log.Fatalf(format, v...)
}

///
/// WarnLogger
///

type WarnLogger struct{}

func (WarnLogger) Println(v ...interface{}) {
	log.Warn(v...)
}

func (WarnLogger) Printf(format string, v ...interface{}) {
	log.Warnf(format, v...)
}

///
/// DebugLogger
///

type DebugLogger struct{}

func (DebugLogger) Println(v ...interface{}) {
	log.Debug(v...)
}

func (DebugLogger) Printf(format string, v ...interface{}) {
	log.Debugf(format, v...)
}
