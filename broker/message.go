package broker

type Binder func() any

// Headers defines message headers as a map of string key-value pairs
type Headers map[string]string

// Message defines the message structure
type Message struct {
	// Headers message headers
	Headers Headers
	// Body message body
	Body any

	// Partition message partition
	Partition int

	// Offset message offset
	Offset int64

	// Msg original/raw message
	Msg any
}

// GetHeaders returns the message headers
func (m Message) GetHeaders() Headers {
	return m.Headers
}

// GetHeader returns the value of a specific header key
func (m Message) GetHeader(key string) string {
	if m.Headers == nil {
		return ""
	}
	return m.Headers[key]
}

// HeadersCopy returns a copy of the message headers
func (m Message) HeadersCopy() Headers {
	if m.Headers == nil {
		return nil
	}
	cp := make(Headers, len(m.Headers))
	for k, v := range m.Headers {
		cp[k] = v
	}
	return cp
}
