package webtransport

type SessionID uint64

type Binder func() any

type ConnectHandler func(SessionID, bool)

type MessageType uint32

type MessagePayload any

type MessageHandler func(SessionID, MessagePayload) error

type HandlerData struct {
	Handler MessageHandler
	Binder  Binder
}

func (h *HandlerData) Create() any {
	if h.Binder != nil {
		return h.Binder()
	}
	return nil
}

type MessageHandlerMap map[MessageType]HandlerData
