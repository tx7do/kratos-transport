package sse

import (
	"bytes"
	"fmt"
	"net/http"
)

const (
	FieldId      = "id"
	FieldData    = "data"
	FieldEvent   = "event"
	FieldRetry   = "retry"
	FieldComment = ":"
)

func containsDoubleNewline(data []byte) (int, int) {
	crCr := bytes.Index(data, []byte("\r\r"))
	lfLf := bytes.Index(data, []byte("\n\n"))
	crLfLf := bytes.Index(data, []byte("\r\n\n"))
	lfCrLf := bytes.Index(data, []byte("\n\r\n"))
	crLfCrLf := bytes.Index(data, []byte("\r\n\r\n"))
	minPos := minPosInt(crCr, minPosInt(lfLf, minPosInt(crLfLf, minPosInt(lfCrLf, crLfCrLf))))
	nLen := 2
	if minPos == crLfCrLf {
		nLen = 4
	} else if minPos == crLfLf || minPos == lfCrLf {
		nLen = 3
	}
	return minPos, nLen
}

func minPosInt(a, b int) int {
	if a < 0 {
		return b
	}
	if b < 0 {
		return a
	}
	if a > b {
		return b
	}
	return a
}

func writeData(w http.ResponseWriter, field string, value []byte) (int, error) {
	return fmt.Fprintf(w, "%s: %s\n", field, value)
}

func writeError(w http.ResponseWriter, message string, status int) {
	http.Error(w, message, status)
}
