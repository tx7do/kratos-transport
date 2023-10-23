package webtransport

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/quic-go/quic-go/http3"
)

type Upgrader struct {
	CheckOrigin func(r *http.Request) bool
}

func (u *Upgrader) Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (http3.HTTPStreamer, error) {
	if r.Method != http.MethodConnect {
		return nil, fmt.Errorf("expected CONNECT request, got %s", r.Method)
	}

	if r.Proto != protocolHeader {
		return nil, fmt.Errorf("unexpected protocol: %s", r.Proto)
	}

	if v, ok := r.Header[webTransportDraftOfferHeaderKey]; !ok || len(v) != 1 || v[0] != "1" {
		return nil, fmt.Errorf("missing or invalid %s header", webTransportDraftOfferHeaderKey)
	}

	checkOrigin := u.CheckOrigin
	if checkOrigin == nil {
		checkOrigin = checkSameOrigin
	}
	if !u.CheckOrigin(r) {
		return nil, errors.New("request origin not allowed")
	}

	w.Header().Add(webTransportDraftHeaderKey, webTransportDraftHeaderValue)
	w.WriteHeader(http.StatusOK)
	w.(http.Flusher).Flush()

	httpStreamer, ok := r.Body.(http3.HTTPStreamer)
	if !ok { // should never happen, unless quic-go changed the API
		return nil, errors.New("failed to take over HTTP qStream")
	}

	return httpStreamer, nil
}
