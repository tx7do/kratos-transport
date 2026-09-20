package tcp

import (
	"context"
	"net/http"

	kratosTransport "github.com/go-kratos/kratos/v2/transport"
)

const (
	KindTcp = "tcp"
)

var _ Transporter = &Transport{}

type Transporter interface {
	kratosTransport.Transporter
	Request() *http.Request
	PathTemplate() string
}

// Transport is a TCP transport.
type Transport struct {
	endpoint  string
	operation string
}

// Kind returns the transport kind.
func (tr *Transport) Kind() kratosTransport.Kind {
	return KindTcp
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
	return nil
}

// RequestHeader returns the request header.
func (tr *Transport) RequestHeader() kratosTransport.Header {
	return nil
}

// ReplyHeader returns the reply header.
func (tr *Transport) ReplyHeader() kratosTransport.Header {
	return nil
}

// PathTemplate returns the http path template.
func (tr *Transport) PathTemplate() string {
	return ""
}

// SetOperation sets the transport operation.
func SetOperation(ctx context.Context, op string) {
	if tr, ok := kratosTransport.FromServerContext(ctx); ok {
		if tr, ok := tr.(*Transport); ok {
			tr.operation = op
		}
	}
}

// headerCarrier is a map-backed kratos header carrier.
type headerCarrier struct {
	h http.Header
}

func (hc *headerCarrier) init() {
	if hc.h == nil {
		hc.h = make(http.Header)
	}
}

// Get returns the value associated with the passed key.
func (hc *headerCarrier) Get(key string) string {
	hc.init()
	return hc.h.Get(key)
}

// Set stores the key-value pair.
func (hc *headerCarrier) Set(key, value string) {
	hc.init()
	hc.h.Set(key, value)
}

// Keys lists the keys stored in this carrier.
func (hc *headerCarrier) Keys() []string {
	hc.init()
	keys := make([]string, 0, len(hc.h))
	for k := range hc.h {
		keys = append(keys, k)
	}
	return keys
}

// Add append value to key-values pair.
func (hc *headerCarrier) Add(key, value string) {
	hc.init()
	hc.h.Add(key, value)
}

// Values returns a slice of values associated with the passed key.
func (hc *headerCarrier) Values(key string) []string {
	hc.init()
	return hc.h.Values(key)
}
