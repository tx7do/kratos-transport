package sse

import (
	"bytes"
	"fmt"
	"net/http"
	"time"
)

// prepareHeaderForSSE writes the default SSE and CORS response headers.
func (s *Server) prepareHeaderForSSE(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	w.Header().Set("Access-Control-Allow-Origin", s.corsAllowOrigin)
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	for k, v := range s.headers {
		w.Header().Set(k, v)
	}
}

// ServeHTTP handles SSE subscription requests and streams events to the client.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Handle OPTIONS preflight by returning CORS headers without auth checks.
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", s.corsAllowOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Token, Last-Event-ID")
		w.Header().Set("Access-Control-Max-Age", "86400")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if s.authorizeFunc != nil {
		token := ""
		if s.tokenExtractor != nil {
			token = s.tokenExtractor(r)
		}

		if err := s.authorizeFunc(r, token); err != nil {
			statusCode := http.StatusUnauthorized
			if isForbidden(err) {
				statusCode = http.StatusForbidden
			}
			writeError(w, err.Error(), statusCode)
			return
		}
	}

	flusher, exist := w.(http.Flusher)
	if !exist {
		writeError(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	s.prepareHeaderForSSE(w)

	streamID := r.URL.Query().Get(s.streamIdKey)
	if streamID == "" {
		writeError(w, "Please specify a stream!", http.StatusInternalServerError)
		return
	}

	stream := s.streamMgr.Get(StreamID(streamID))
	if stream == nil {
		if !s.autoStream {
			writeError(w, "Stream not found!", http.StatusInternalServerError)
			return
		}

		stream = s.CreateStream(StreamID(streamID))
	}

	eventId := ""
	if id := r.Header.Get("Last-Event-ID"); id != "" {
		eventId = id
	}

	sub := stream.addSubscriber(eventId, r)

	go func() {
		<-r.Context().Done()

		sub.close()

		if s.autoStream && !s.autoReplay && stream.getSubscriberCount() == 0 {
			s.streamMgr.RemoveWithID(StreamID(streamID))
		}
	}()

	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	for ev := range sub.connection {
		// 仅当事件完全为空（显式关闭哨兵）才断流；
		// 只带 Retry/ID 的元数据事件是合法载荷，此前会误断客户端
		if len(ev.Data) == 0 && len(ev.Comment) == 0 && len(ev.Retry) == 0 && len(ev.ID) == 0 && len(ev.Event) == 0 {
			break
		}

		if s.eventTTL != 0 && time.Now().After(ev.timestamp.Add(s.eventTTL)) {
			continue
		}

		if len(ev.Data) > 0 {
			_, _ = writeData(w, FieldId, ev.ID)

			if s.splitData {
				sd := bytes.Split(ev.Data, []byte("\n"))
				for i := range sd {
					_, _ = writeData(w, FieldData, sd[i])
				}
			} else {
				if bytes.HasPrefix(ev.Data, []byte(":")) {
					_, _ = fmt.Fprintf(w, "%s\n", ev.Data)
				} else {
					_, _ = writeData(w, FieldData, ev.Data)
				}
			}

			if len(ev.Event) > 0 {
				_, _ = writeData(w, FieldEvent, ev.Event)
			}

			if len(ev.Retry) > 0 {
				_, _ = writeData(w, FieldRetry, ev.Retry)
			}
		}

		if len(ev.Comment) > 0 {
			_, _ = writeData(w, "", ev.Comment)
		}

		_, _ = fmt.Fprint(w, "\n")

		flusher.Flush()
	}
}
