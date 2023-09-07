package asynq

type MessagePayload any

type Binder func() any

type MessageHandler func(string, MessagePayload) error

type HandlerData struct {
	Handler MessageHandler
	Binder  Binder
}
type MessageHandlerMap map[string]HandlerData
