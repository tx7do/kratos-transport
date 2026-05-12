package sse

import (
	"errors"
	"net/http"
	"strings"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)

type TokenExtractor func(r *http.Request) string

type AuthorizeFunc func(r *http.Request, token string) error

func DefaultTokenExtractor(r *http.Request) string {
	if r == nil {
		return ""
	}

	if token := extractBearerToken(r.Header.Get("Authorization")); token != "" {
		return token
	}
	if token := strings.TrimSpace(r.Header.Get("X-Token")); token != "" {
		return token
	}
	if token := strings.TrimSpace(r.URL.Query().Get("token")); token != "" {
		return token
	}

	return ""
}

func isForbidden(err error) bool {
	return errors.Is(err, ErrForbidden)
}

func extractBearerToken(auth string) string {
	auth = strings.TrimSpace(auth)
	if auth == "" {
		return ""
	}
	if len(auth) >= 7 && strings.EqualFold(auth[:7], "Bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return auth
}
