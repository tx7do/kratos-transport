package kcp

import "github.com/go-kratos/kratos/v2/encoding"

type ClientOption func(o *Client)

func WithClientCodec(codec string) ClientOption {
	return func(c *Client) {
		c.codec = encoding.GetCodec(codec)
	}
}

func WithEndpoint(uri string) ClientOption {
	return func(c *Client) {
		c.url = uri
	}
}

func WithClientRawDataHandler(h ClientRawMessageHandler) ClientOption {
	return func(c *Client) {
		c.rawMessageHandler = h
	}
}

func WithClientBlockCrypt(password, salt string) ClientOption {
	return func(c *Client) {
		c.blockCryptPassword = password
		c.blockCryptSalt = salt
	}
}

func WithClientDataShards(dataShards int) ClientOption {
	return func(c *Client) {
		c.dataShards = dataShards
	}
}

func WithClientParityShards(parityShards int) ClientOption {
	return func(c *Client) {
		c.parityShards = parityShards
	}
}
