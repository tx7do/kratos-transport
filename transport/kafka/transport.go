package kafka

import (
	"github.com/go-kratos/kratos/v2/selector"
	"github.com/go-kratos/kratos/v2/transport"
	KAFKA "github.com/segmentio/kafka-go"
)

const (
	KindKafka transport.Kind = "kafka"
)

var _ transport.Transporter = &Transport{}

// Transport is a Kafka transport.
type Transport struct {
	endpoint    string
	operation   string
	reqHeader   headerCarrier
	replyHeader headerCarrier
	nodeFilters []selector.NodeFilter
}

// Kind returns the transport kind.
func (tr *Transport) Kind() transport.Kind {
	return KindKafka
}

// Endpoint returns the transport endpoint.
func (tr *Transport) Endpoint() string {
	return tr.endpoint
}

// Operation returns the transport operation.
func (tr *Transport) Operation() string {
	return tr.operation
}

// RequestHeader returns the request header.
func (tr *Transport) RequestHeader() transport.Header {
	return tr.reqHeader
}

// ReplyHeader returns the reply header.
func (tr *Transport) ReplyHeader() transport.Header {
	return tr.replyHeader
}

// NodeFilters returns the client select filters.
func (tr *Transport) NodeFilters() []selector.NodeFilter {
	return tr.nodeFilters
}

type headerCarrier KAFKA.Header

// Get returns the value associated with the passed key.
func (mc headerCarrier) Get(_ string) string {
	return ""
}

// Set stores the key-value pair.
func (mc headerCarrier) Set(_ string, _ string) {
}

// Keys lists the keys stored in this carrier.
func (mc headerCarrier) Keys() []string {
	return nil
}
