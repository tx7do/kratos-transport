package broker

import (
	"context"
	"fmt"
	"reflect"
)

// Binder defines a function that returns an instance for binding message body
type Binder func() any

// Creator is a generic function that creates a new instance of type T
func Creator[T any]() *T {
	var t T
	return &t
}

// Headers defines message headers as a map of string key-value pairs
type Headers map[string]string

// Metadata defines message metadata as a map of string key-value pairs
type Metadata map[string]any

// Message defines the message structure
type Message struct {
	// ID message ID
	ID string

	// Headers message headers
	Headers Headers

	// Body message body
	Body any

	// Key message key (Kafka Key / RocketMQ Key / RabbitMQ RoutingKey...)
	Key string

	// Metadata can hold additional metadata about the message
	Metadata Metadata

	// Partition message partition
	Partition int
	// Offset message offset
	Offset int64

	// Msg original/raw message (kafka.Message, amqp.Delivery...)
	Msg any
}

// SetHeader sets a header key-value pair
func (m *Message) SetHeader(key, value string) {
	if m.Headers == nil {
		m.Headers = make(map[string]string)
	}
	m.Headers[key] = value
}

// WithHeader sets a header key-value pair and returns the message for chaining
func (m *Message) WithHeader(key, value string) *Message {
	m.SetHeader(key, value)
	return m
}

// GetHeader returns the value of a specific header key
func (m *Message) GetHeader(key string) string {
	if m.Headers == nil {
		return ""
	}
	return m.Headers[key]
}

// HeadersCopy returns a copy of the message headers
func (m *Message) HeadersCopy() Headers {
	if m.Headers == nil {
		return nil
	}
	cp := make(Headers, len(m.Headers))
	for k, v := range m.Headers {
		cp[k] = v
	}
	return cp
}

// MergeHeaders merges the provided headers into the message headers
func (m *Message) MergeHeaders(hdrs Headers) {
	if hdrs == nil {
		return
	}
	for k, v := range hdrs {
		m.SetHeader(k, v)
	}
}

// SetMetadata sets a metadata key-value pair
func (m *Message) SetMetadata(key string, value any) {
	if m.Metadata == nil {
		m.Metadata = make(map[string]any)
	}
	m.Metadata[key] = value
}

// WithMetadata sets a metadata key-value pair and returns the message for chaining
func (m *Message) WithMetadata(key string, value any) *Message {
	m.SetMetadata(key, value)
	return m
}

// AckSuccess acknowledges the message if the underlying Msg supports it
func (m *Message) AckSuccess() error {
	if ack, ok := m.Msg.(interface{ Ack() error }); ok {
		return ack.Ack()
	}
	return nil
}

// BodyBytes returns the message body as a byte slice if possible
func (m *Message) BodyBytes() []byte {
	if m.Body == nil {
		return nil
	}

	if b, ok := m.Body.([]byte); ok {
		return b
	}
	return nil
}

// WithBody sets the message body and returns the message for chaining
func (m *Message) WithBody(body any) *Message {
	m.Body = body
	return m
}

// WithKey sets the message key and returns the message for chaining
func (m *Message) WithKey(key string) *Message {
	m.Key = key
	return m
}

// SetDelay sets the delay level for the message
func (m *Message) SetDelay(level string) {
	m.SetMetadata("x-delay-level", level)
}

// Clone creates a shallow copy of the message
func (m *Message) Clone() *Message {
	return &Message{
		ID:        m.ID,
		Headers:   m.HeadersCopy(),
		Body:      m.Body,
		Key:       m.Key,
		Metadata:  m.MetadataCopy(),
		Partition: m.Partition,
		Offset:    m.Offset,
		Msg:       m.Msg,
	}
}

// MetadataCopy returns a copy of the message metadata
func (m *Message) MetadataCopy() Metadata {
	if m.Metadata == nil {
		return nil
	}
	cp := make(Metadata, len(m.Metadata))
	for k, v := range m.Metadata {
		cp[k] = v
	}
	return cp
}

// GetMetadata retrieves the value of a specific metadata key
func (m *Message) GetMetadata(key string) any {
	if m.Metadata == nil {
		return nil
	}
	return m.Metadata[key]
}

type metadataKey struct{}

// ExtractContext extracts the message metadata into the provided context
func (m *Message) ExtractContext(baseCtx context.Context) context.Context {
	if m.Metadata == nil {
		return baseCtx
	}
	return context.WithValue(baseCtx, metadataKey{}, m.Metadata)
}

// GetMetadataFromContext retrieves the message metadata from the context
func GetMetadataFromContext(ctx context.Context) Metadata {
	if md, ok := ctx.Value(metadataKey{}).(Metadata); ok {
		return md
	}
	return nil
}

// InjectContext injects values from the context into the message headers based on the provided keys
func (m *Message) InjectContext(ctx context.Context, keys ...any) *Message {
	for _, key := range keys {
		if val := ctx.Value(key); val != nil {
			m.SetHeader(formatKey(key), fmt.Sprintf("%v", val))
		}
	}
	return m
}

// ExtractToContext extracts specified headers from the message into the context based on the provided key mapping
func (m *Message) ExtractToContext(baseCtx context.Context, keyMapping map[string]any) context.Context {
	ctx := baseCtx
	for hKey, cKey := range keyMapping {
		if val := m.GetHeader(hKey); val != "" {
			ctx = context.WithValue(ctx, cKey, val)
		}
	}
	return ctx
}

// formatKey formats the key into a string representation
func formatKey(key any) string {
	if key == nil {
		return ""
	}

	if s, ok := key.(string); ok {
		return s
	}

	if s, ok := key.(fmt.Stringer); ok {
		return s.String()
	}

	t := reflect.TypeOf(key)
	if t.Kind() == reflect.Struct || t.Kind() == reflect.Ptr {
		return t.String()
	}

	return fmt.Sprintf("%v", key)
}

type MessageOption func(*Message)

// NewMessage creates a new Message with the provided body and options
func NewMessage(body any, opts ...MessageOption) *Message {
	m := &Message{
		Body:      body,
		Headers:   make(Headers),
		Metadata:  make(Metadata),
		Partition: -1,
		Offset:    -1,
	}
	for _, o := range opts {
		if o != nil {
			o(m)
		}
	}
	return m
}

// NewMessageWithContext creates a new Message and injects context values into its headers
func NewMessageWithContext(ctx context.Context, body any, keys ...any) *Message {
	return NewMessage(body).InjectContext(ctx, keys...)
}

// WithID set the message ID
func WithID(id string) MessageOption {
	return func(m *Message) { m.ID = id }
}

// WithKey set the message key
func WithKey(key string) MessageOption {
	return func(m *Message) { m.Key = key }
}

// WithMsg set the original/raw message
func WithMsg(msg any) MessageOption {
	return func(m *Message) { m.Msg = msg }
}

// WithPartitionOffset set the Partition and Offset of the Message
func WithPartitionOffset(partition int, offset int64) MessageOption {
	return func(m *Message) {
		m.Partition = partition
		m.Offset = offset
	}
}

// WithHeaders will copy the provided headers into Message.Headers (no-op if nil)
func WithHeaders(h Headers) MessageOption {
	return func(m *Message) {
		if h == nil {
			return
		}
		if m.Headers == nil {
			m.Headers = make(Headers, len(h))
		}
		for k, v := range h {
			m.Headers[k] = v
		}
	}
}

// WithHeader add single header key-value pair
func WithHeader(key, value string) MessageOption {
	return func(m *Message) {
		m.SetHeader(key, value)
	}
}

// WithMetadata will copy the provided metadata into Message.Metadata (no-op if nil)
func WithMetadata(md Metadata) MessageOption {
	return func(m *Message) {
		if md == nil {
			return
		}
		if m.Metadata == nil {
			m.Metadata = make(Metadata, len(md))
		}
		for k, v := range md {
			m.Metadata[k] = v
		}
	}
}

// WithMetadataKV add single metadata key-value pair
func WithMetadataKV(key string, value any) MessageOption {
	return func(m *Message) {
		m.SetMetadata(key, value)
	}
}
