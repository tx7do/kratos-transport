package websocket

import "github.com/go-kratos/kratos/v2/encoding"

type ClientOption func(o *Client)

func WithClientCodec(c string) ClientOption {
	return func(o *Client) {
		o.codec = encoding.GetCodec(c)
	}
}

func WithEndpoint(uri string) ClientOption {
	return func(o *Client) {
		o.url = uri
	}
}

func WithClientPayloadType(payloadType PayloadType) ClientOption {
	return func(c *Client) {
		c.payloadType = payloadType
	}
}
