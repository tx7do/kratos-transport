package websocket

import (
	"context"
	"net/http"

	"github.com/go-kratos/kratos/v2/transport"
)

const (
	KindWebsocket transport.Kind = "websocket"
)

var _ Transporter = &Transport{}

type Transporter interface {
	transport.Transporter
	Request() *http.Request
	PathTemplate() string
}

// Transport is a websocket transport.
type Transport struct {
	endpoint     string
	operation    string
	reqHeader    headerCarrier
	replyHeader  headerCarrier
	request      *http.Request
	pathTemplate string
}

// Kind returns the transport kind.
func (tr *Transport) Kind() transport.Kind {
	return KindWebsocket
}

// Endpoint returns the transport endpoint.
func (tr *Transport) Endpoint() string {
	return tr.endpoint
}

// Operation returns the transport operation.
func (tr *Transport) Operation() string {
	return tr.operation
}

// Request returns the HTTP request.
func (tr *Transport) Request() *http.Request {
	return tr.request
}

// RequestHeader returns the request header.
func (tr *Transport) RequestHeader() transport.Header {
	return tr.reqHeader
}

// ReplyHeader returns the reply header.
func (tr *Transport) ReplyHeader() transport.Header {
	return tr.replyHeader
}

// PathTemplate returns the http path template.
func (tr *Transport) PathTemplate() string {
	return tr.pathTemplate
}

// SetOperation sets the transport operation.
func SetOperation(ctx context.Context, op string) {
	if tr, ok := transport.FromServerContext(ctx); ok {
		if tr, ok := tr.(*Transport); ok {
			tr.operation = op
		}
	}
}

type headerCarrier http.Header

// Get returns the value associated with the passed key.
func (hc headerCarrier) Get(key string) string {
	return http.Header(hc).Get(key)
}

// Set stores the key-value pair.
func (hc headerCarrier) Set(key, value string) {
	http.Header(hc).Set(key, value)
}

// Keys lists the keys stored in this carrier.
func (hc headerCarrier) Keys() []string {
	keys := make([]string, 0, len(hc))
	for k := range http.Header(hc) {
		keys = append(keys, k)
	}
	return keys
}

// Add append value to key-values pair.
func (hc headerCarrier) Add(key string, value string) {
	http.Header(hc).Add(key, value)
}

// Values returns a slice of values associated with the passed key.
func (hc headerCarrier) Values(key string) []string {
	return http.Header(hc).Values(key)
}
