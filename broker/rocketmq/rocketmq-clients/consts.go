package rocketmqClients

import "time"

const (
	// maximum waiting time for receive func
	defaultAwaitDuration = time.Second * 5

	// maximum number of messages received at one time
	defaultMaxMessageNum int32 = 16

	// invisibleDuration should > 20s
	defaultInvisibleDuration = time.Second * 20

	defaultReceiveInterval = time.Second * 3
)
