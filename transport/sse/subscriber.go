package sse

import (
	"net/http"
	"net/url"
	"strings"
)

type Subscriber struct {
	quit       chan *Subscriber
	connection chan *Event
	removed    chan struct{}
	eventId    int
	URL        *url.URL
	Header     http.Header
}

func (s *Subscriber) close() {
	s.quit <- s
	if s.removed != nil {
		<-s.removed
	}
}

func (s *Subscriber) HeaderValue(key string) string {
	if s == nil || s.Header == nil {
		return ""
	}
	return s.Header.Get(key)
}

func (s *Subscriber) Authorization() string {
	return s.HeaderValue("Authorization")
}

func (s *Subscriber) BearerToken() string {
	auth := strings.TrimSpace(s.Authorization())
	if auth == "" {
		return ""
	}

	if len(auth) >= 7 && strings.EqualFold(auth[:7], "Bearer ") {
		return strings.TrimSpace(auth[7:])
	}

	return auth
}

func (s *Subscriber) Token(headerKey string) string {
	if headerKey != "" {
		if token := strings.TrimSpace(s.HeaderValue(headerKey)); token != "" {
			return token
		}
	}

	if token := s.BearerToken(); token != "" {
		return token
	}

	if s != nil && s.URL != nil {
		return strings.TrimSpace(s.URL.Query().Get("token"))
	}

	return ""
}
