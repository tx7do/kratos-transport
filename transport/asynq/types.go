package asynq

type MessagePayload any

type Creator func() any

type MessageHandler func(string, MessagePayload) error

type MessageHandlerData struct {
	Handler MessageHandler
	Creator Creator
}

func (h *MessageHandlerData) Create() any {
	if h.Creator != nil {
		return h.Creator()
	}
	return nil
}

type MessageHandlerMap map[string]MessageHandlerData
