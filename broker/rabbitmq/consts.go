package rabbitmq

import "time"

const (
	defaultMinResubscribeDelay = 100 * time.Millisecond
	defaultMaxResubscribeDelay = 30 * time.Second
	defaultExpFactor           = time.Duration(2)
	defaultResubscribeDelay    = defaultMinResubscribeDelay
)
