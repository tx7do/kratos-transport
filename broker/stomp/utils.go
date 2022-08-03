package stomp

import "github.com/go-stomp/stomp/v3/frame"

func stompHeaderToMap(h *frame.Header) map[string]string {
	m := map[string]string{}
	for i := 0; i < h.Len(); i++ {
		k, v := h.GetAt(i)
		m[k] = v
	}
	return m
}
