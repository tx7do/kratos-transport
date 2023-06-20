package sse

import (
	"bufio"
	"context"
	"encoding/base64"
	"io"
	"time"
)

type Event struct {
	timestamp time.Time
	ID        []byte
	Data      []byte
	Event     []byte
	Retry     []byte
	Comment   []byte
}

func (e *Event) hasContent() bool {
	return len(e.ID) > 0 || len(e.Data) > 0 || len(e.Event) > 0 || len(e.Retry) > 0
}

func (e *Event) encodeBase64() {
	dataLen := len(e.Data)
	if dataLen > 0 {
		output := make([]byte, base64.StdEncoding.EncodedLen(dataLen))
		base64.StdEncoding.Encode(output, e.Data)
		e.Data = output
	}
}

type EventStreamReader struct {
	scanner *bufio.Scanner
}

func NewEventStreamReader(eventStream io.Reader, maxBufferSize int) *EventStreamReader {
	scanner := bufio.NewScanner(eventStream)
	initBufferSize := minPosInt(4096, maxBufferSize)
	scanner.Buffer(make([]byte, initBufferSize), maxBufferSize)

	split := func(data []byte, atEOF bool) (int, []byte, error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}

		if i, nLen := containsDoubleNewline(data); i >= 0 {
			return i + nLen, data[0:i], nil
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	}
	scanner.Split(split)

	return &EventStreamReader{
		scanner: scanner,
	}
}

func (e *EventStreamReader) ReadEvent() ([]byte, error) {
	if e.scanner.Scan() {
		event := e.scanner.Bytes()
		return event, nil
	}
	if err := e.scanner.Err(); err != nil {
		if err == context.Canceled {
			return nil, io.EOF
		}
		return nil, err
	}
	return nil, io.EOF
}
