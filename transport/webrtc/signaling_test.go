package webrtc

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pion/webrtc/v4"
)

func TestDecodeSignalRequest_WrappedOffer(t *testing.T) {
	body := []byte(`{"offer":{"type":"offer","sdp":"v=0\r\n"}}`)
	req := httptest.NewRequest(http.MethodPost, "/signal", bytes.NewReader(body))

	var offer webrtc.SessionDescription
	if err := decodeSignalRequest(req, &offer); err != nil {
		t.Fatalf("decodeSignalRequest() error = %v", err)
	}
	if offer.Type != webrtc.SDPTypeOffer {
		t.Fatalf("unexpected offer type: %v", offer.Type)
	}
	if offer.SDP == "" {
		t.Fatal("expected SDP not empty")
	}
}

func TestDecodeSignalRequest_DirectOffer(t *testing.T) {
	body := []byte(`{"type":"offer","sdp":"v=0\r\n"}`)
	req := httptest.NewRequest(http.MethodPost, "/signal", bytes.NewReader(body))

	var offer webrtc.SessionDescription
	if err := decodeSignalRequest(req, &offer); err != nil {
		t.Fatalf("decodeSignalRequest() error = %v", err)
	}
	if offer.Type != webrtc.SDPTypeOffer {
		t.Fatalf("unexpected offer type: %v", offer.Type)
	}
}

func TestDecodeSignalRequest_Invalid(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/signal", bytes.NewReader([]byte(`{"offer":{}}`)))

	var offer webrtc.SessionDescription
	if err := decodeSignalRequest(req, &offer); err == nil {
		t.Fatal("expected error for invalid offer")
	}
}

func TestEncodeSignalResponse_Nil(t *testing.T) {
	rr := httptest.NewRecorder()
	if err := encodeSignalResponse(rr, nil); err == nil {
		t.Fatal("expected error for nil response")
	}
}

func TestWriteSignalError(t *testing.T) {
	rr := httptest.NewRecorder()
	if err := writeSignalError(rr, errors.New("boom")); err != nil {
		t.Fatalf("writeSignalError() error = %v", err)
	}
	if rr.Body.Len() == 0 {
		t.Fatal("expected non-empty body")
	}
}
