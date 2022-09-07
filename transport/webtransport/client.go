package webtransport

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/http3"
	"github.com/lucas-clemente/quic-go/quicvarint"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	transport *http3.RoundTripper

	ctx       context.Context
	ctxCancel context.CancelFunc

	timeout time.Duration
	tlsConf *tls.Config

	url string

	sessions *sessionManager
	session  *Session
}

func NewClient(opts ...ClientOption) *Client {
	cli := &Client{
		transport: &http3.RoundTripper{},
	}
	cli.init(opts...)
	return cli
}

func (c *Client) init(opts ...ClientOption) {
	for _, o := range opts {
		o(c)
	}

	c.ctx, c.ctxCancel = context.WithCancel(context.Background())

	timeout := c.timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	c.sessions = newSessionManager(timeout)

	if c.tlsConf == nil {
		c.tlsConf = &tls.Config{
			RootCAs:            generateCertPool(),
			InsecureSkipVerify: true,
		}
	}
	c.transport.TLSClientConfig = c.tlsConf

	c.transport.EnableDatagrams = true

	if c.transport.AdditionalSettings == nil {
		c.transport.AdditionalSettings = make(map[uint64]uint64)
	}

	c.transport.StreamHijacker = func(ft http3.FrameType, conn quic.Connection, str quic.Stream, e error) (hijacked bool, err error) {
		if isWebTransportError(e) {
			return true, nil
		}
		if ft != webTransportFrameType {
			return false, nil
		}
		id, err := quicvarint.Read(quicvarint.NewReader(str))
		if err != nil {
			if isWebTransportError(err) {
				return true, nil
			}
			return false, err
		}
		c.sessions.AddStream(conn, str, SessionID(id))
		return true, nil
	}
	c.transport.UniStreamHijacker = func(st http3.StreamType, conn quic.Connection, str quic.ReceiveStream, err error) (hijacked bool) {
		if st != webTransportUniStreamType && !isWebTransportError(err) {
			return false
		}
		c.sessions.AddUniStream(conn, str)
		return true
	}
	if c.transport.QuicConfig == nil {
		c.transport.QuicConfig = &quic.Config{}
	}
	if c.transport.QuicConfig.MaxIncomingStreams == 0 {
		c.transport.QuicConfig.MaxIncomingStreams = 100
	}
}

func (c *Client) Start() error {
	u, err := url.Parse(c.url)
	if err != nil {
		return err
	}

	reqHdr := http.Header{}
	reqHdr.Add(webTransportDraftOfferHeaderKey, "1")

	req := &http.Request{
		Method: http.MethodConnect,
		Header: reqHdr,
		Proto:  "webtransport",
		Host:   u.Host,
		URL:    u,
	}
	req = req.WithContext(c.ctx)

	rsp, err := c.transport.RoundTripOpt(req,
		http3.RoundTripOpt{DontCloseRequestStream: true},
	)
	if err != nil {
		return err
	}
	if rsp.StatusCode < 200 || rsp.StatusCode >= 300 {
		return fmt.Errorf("received status %d", rsp.StatusCode)
	}

	str := rsp.Body.(http3.HTTPStreamer).HTTPStream()
	session := c.sessions.AddSession(
		rsp.Body.(http3.Hijacker).StreamCreator(),
		SessionID(str.StreamID()),
		str,
	)

	c.session = session

	log.Infof("[webtransport] client connected to: %s", c.url)

	return nil
}

func (c *Client) Stop() error {
	log.Info("[webtransport] client stopping")
	return nil
}
