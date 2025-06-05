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
type MessageHandlerMap map[MessageType]HandlerData
