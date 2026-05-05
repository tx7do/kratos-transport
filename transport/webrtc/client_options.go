package webrtc

import (
	"time"

	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/pion/webrtc/v4"
)

type ClientOption func(o *Client)

func WithSignalURL(uri string) ClientOption {
	return func(c *Client) {
		c.signalURL = uri
	}
}

func WithAuthorization(token string) ClientOption {
	return func(c *Client) {
		c.authorization = token
	}
}

func WithClientCodec(name string) ClientOption {
	return func(c *Client) {
		if name != "" {
			c.codec = encoding.GetCodec(name)
		}
	}
}

func WithClientPayloadType(payloadType PayloadType) ClientOption {
	return func(c *Client) {
		c.payloadType = payloadType
	}
}

func WithClientDataChannelLabel(label string) ClientOption {
	return func(c *Client) {
		if label != "" {
			c.dataChannelLabel = label
		}
	}
}

func WithClientWebRTCConfiguration(cfg webrtc.Configuration) ClientOption {
	return func(c *Client) {
		c.webrtcConfig = cfg
	}
}

func WithClientConnectTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		if timeout > 0 {
			c.connectTimeout = timeout
		}
	}
}

func WithClientSignalTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		if timeout > 0 {
			c.signalTimeout = timeout
		}
	}
}
