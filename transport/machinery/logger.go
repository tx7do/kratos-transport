package machinery

import (
	"errors"
	"fmt"
	"github.com/RichardKnop/logging"
	"github.com/go-kratos/kratos/v2/log"
	"os"
)

const (
	logKey = "machinery"
)

type logger struct {
	level log.Level
}

func newLogger(level log.Level) logging.LoggerInterface {
	return &logger{
		level: level,
	}
}

func (l *logger) Print(args ...interface{}) {
	_ = log.GetLogger().Log(l.level, logKey, fmt.Sprint(args...))
}
func (l *logger) Printf(format string, args ...interface{}) {
	_ = log.GetLogger().Log(l.level, logKey, fmt.Sprintf(format, args...))
}
func (l *logger) Println(args ...interface{}) {
	_ = log.GetLogger().Log(l.level, logKey, fmt.Sprint(args...))
}

func (l *logger) Fatal(args ...interface{}) {
	_ = log.GetLogger().Log(l.level, logKey, fmt.Sprint(args...))
	os.Exit(1)
}
func (l *logger) Fatalf(format string, args ...interface{}) {
	_ = log.GetLogger().Log(l.level, logKey, fmt.Sprintf(format, args...))
	os.Exit(1)
}
func (l *logger) Fatalln(args ...interface{}) {
	_ = log.GetLogger().Log(l.level, logKey, fmt.Sprint(args...))
	os.Exit(1)
}

func (l *logger) Panic(args ...interface{}) {
	_ = log.GetLogger().Log(l.level, logKey, fmt.Sprint(args...))
	panic(errors.New(fmt.Sprint(args...)))
}
func (l *logger) Panicf(format string, args ...interface{}) {
	_ = log.GetLogger().Log(l.level, logKey, fmt.Sprintf(format, args...))
	panic(errors.New(fmt.Sprintf(format, args...)))
}
func (l *logger) Panicln(args ...interface{}) {
	_ = log.GetLogger().Log(l.level, logKey, fmt.Sprint(args...))
	panic(errors.New(fmt.Sprint(args...)))
}
