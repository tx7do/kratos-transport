package sse

type ClientOption func(o *Client)

func WithEndpoint(uri string) ClientOption {
	return func(c *Client) {
		c.url = uri
	}
}
