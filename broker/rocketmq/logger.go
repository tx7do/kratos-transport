package rocketmq

import "github.com/go-kratos/kratos/v2/log"

type logger struct {
}

func (l logger) Debug(msg string, fields map[string]interface{}) {
	log.Debug(msg)
}

func (l logger) Info(msg string, fields map[string]interface{}) {
	log.Info(msg)
}

func (l logger) Warning(msg string, fields map[string]interface{}) {
	log.Warn(msg)
}

func (l logger) Error(msg string, fields map[string]interface{}) {
	log.Error(msg)
}

func (l logger) Fatal(msg string, fields map[string]interface{}) {
	log.Fatal(msg)
}

func (l logger) Level(level string) {
}

func (l logger) OutputPath(path string) (err error) {
	return nil
}
