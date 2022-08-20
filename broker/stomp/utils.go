package stomp

import (
	"github.com/go-stomp/stomp/v3/frame"
	"regexp"
)

var re = regexp.MustCompile("^stomp(\\+ssl)?://.*")

func hasUrlPrefix(url string) bool {
	return re.MatchString(url)
}

func isSchema(schema string) bool {
	return schema == "stomp" || schema == "stomp+ssl"
}

func refitUrl(url string, enableTLS bool) string {
	if !hasUrlPrefix(url) {
		prefix := "stomp://"
		if enableTLS {
			prefix = "stomp+ssl://"
		}
		url = prefix + url
	}
	return url
}

func stompHeaderToMap(h *frame.Header) map[string]string {
	m := map[string]string{}
	for i := 0; i < h.Len(); i++ {
		k, v := h.GetAt(i)
		m[k] = v
	}
	return m
}
