package webtransport

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

const sessionCloseErrorCode quic.StreamErrorCode = 0x170d7b68

type SendStream interface {
	io.Writer
	io.Closer

	CancelWrite(ErrorCode)

	SetWriteDeadline(time.Time) error
}

type ReceiveStream interface {
	io.Reader
	CancelRead(ErrorCode)

	SetReadDeadline(time.Time) error
}

type Stream interface {
	SendStream
	ReceiveStream

	SetDeadline(time.Time) error
}

type sendStream struct {
	qStream quic.SendStream
	// WebTransport qStream header.
	// Set by the constructor, set to nil once sent out.
	// Might be initialized to nil if this sendStream is part of an incoming bidirectional qStream.
	streamHdr []byte

	onClose func()
}

var _ SendStream = &sendStream{}

func newSendStream(qStream quic.SendStream, hdr []byte, onClose func()) *sendStream {
	return &sendStream{qStream: qStream, streamHdr: hdr, onClose: onClose}
}

func (s *sendStream) maybeSendStreamHeader() error {
	if len(s.streamHdr) == 0 {
		return nil
	}
	if _, err := s.qStream.Write(s.streamHdr); err != nil {
		return err
	}
	s.streamHdr = nil
	return nil
}

func (s *sendStream) Write(b []byte) (int, error) {
	if err := s.maybeSendStreamHeader(); err != nil {
		return 0, err
	}
	n, err := s.qStream.Write(b)
	if err != nil && !isTimeoutError(err) {
		s.onClose()
	}
	return n, maybeConvertStreamError(err)
}

func (s *sendStream) CancelWrite(e ErrorCode) {
	s.qStream.CancelWrite(webtransportCodeToHTTPCode(e))
	s.onClose()
}

func (s *sendStream) closeWithSession() {
	s.qStream.CancelWrite(sessionCloseErrorCode)
}

func (s *sendStream) Close() error {
	if err := s.maybeSendStreamHeader(); err != nil {
		return err
	}
	s.onClose()
	return maybeConvertStreamError(s.qStream.Close())
}

func (s *sendStream) SetWriteDeadline(t time.Time) error {
	return maybeConvertStreamError(s.qStream.SetWriteDeadline(t))
}

type receiveStream struct {
	qStream quic.ReceiveStream
	onClose func()
}

var _ ReceiveStream = &receiveStream{}

func newReceiveStream(qStream quic.ReceiveStream, onClose func()) *receiveStream {
	return &receiveStream{qStream: qStream, onClose: onClose}
}

func (s *receiveStream) Read(b []byte) (int, error) {
	n, err := s.qStream.Read(b)
	if err != nil && !isTimeoutError(err) {
		s.onClose()
	}
	return n, maybeConvertStreamError(err)
}

func (s *receiveStream) CancelRead(ec ErrorCode) {
	s.qStream.CancelRead(webtransportCodeToHTTPCode(ec))
	s.onClose()
}

func (s *receiveStream) closeWithSession() {
	s.qStream.CancelRead(sessionCloseErrorCode)
}

func (s *receiveStream) SetReadDeadline(t time.Time) error {
	return maybeConvertStreamError(s.qStream.SetReadDeadline(t))
}

type stream struct {
	*sendStream
	*receiveStream

	mx                             sync.Mutex
	sendSideClosed, recvSideClosed bool
	onClose                        func()
}

var _ Stream = &stream{}

func newStream(qStream quic.Stream, hdr []byte, onClose func()) *stream {
	s := &stream{onClose: onClose}
	s.sendStream = newSendStream(qStream, hdr, func() { s.registerClose(true) })
	s.receiveStream = newReceiveStream(qStream, func() { s.registerClose(false) })
	return s
}

func (s *stream) registerClose(isSendSide bool) {
	s.mx.Lock()
	if isSendSide {
		s.sendSideClosed = true
	} else {
		s.recvSideClosed = true
	}
	isClosed := s.sendSideClosed && s.recvSideClosed
	s.mx.Unlock()

	if isClosed {
		s.onClose()
	}
}

func (s *stream) closeWithSession() {
	s.sendStream.closeWithSession()
	s.receiveStream.closeWithSession()
}

func (s *stream) SetDeadline(t time.Time) error {
	// TODO: implement
	return nil
	// return maybeConvertStreamError(s.SendStream.SetDeadline(t))
}

func maybeConvertStreamError(err error) error {
	if err == nil {
		return nil
	}
	var streamErr *quic.StreamError
	if errors.As(err, &streamErr) {
		errorCode, cErr := httpCodeToWebtransportCode(streamErr.ErrorCode)
		if cErr != nil {
			return fmt.Errorf("qStream reset, but failed to convert qStream error %d: %w", streamErr.ErrorCode, cErr)
		}
		return &StreamError{ErrorCode: errorCode}
	}
	return err
}

func isTimeoutError(err error) bool {
	netErr, ok := err.(net.Error)
	if !ok {
		return false
	}
	return netErr.Timeout()
}
