package trpc

import (
	trpcClient "trpc.group/trpc-go/trpc-go/client"
)

type Client struct {
	Client trpcClient.Client

	err error
}

func NewClient(opts ...ClientOption) *Client {
	srv := &Client{
		Client: trpcClient.New(),
	}

	srv.init(opts...)

	return srv
}

func (c *Client) init(opts ...ClientOption) {
	for _, o := range opts {
		o(c)
	}
}
