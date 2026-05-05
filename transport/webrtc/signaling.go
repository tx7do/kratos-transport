package webrtc

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/pion/webrtc/v4"
)

type signalRequest struct {
	Offer *webrtc.SessionDescription `json:"offer"`
}

type signalResponse struct {
	Answer    webrtc.SessionDescription `json:"answer"`
	SessionID SessionID                 `json:"session_id,omitempty"`
}

type signalError struct {
	Error string `json:"error"`
}

func decodeSignalRequest(req *http.Request, offer *webrtc.SessionDescription) error {
	if req == nil || req.Body == nil {
		return errors.New("empty request body")
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return errors.New("empty request body")
	}

	var wrapped signalRequest
	if err = json.Unmarshal(body, &wrapped); err == nil && wrapped.Offer != nil {
		*offer = *wrapped.Offer
		if offer.Type == 0 || offer.SDP == "" {
			return errors.New("invalid offer")
		}
		return nil
	}

	if err = json.Unmarshal(body, offer); err != nil {
		return err
	}
	if offer.Type == 0 || offer.SDP == "" {
		return errors.New("invalid offer")
	}
	return nil
}

func encodeSignalResponse(w http.ResponseWriter, resp *signalResponse) error {
	if resp == nil {
		return errors.New("nil response")
	}
	return json.NewEncoder(w).Encode(resp)
}

func writeSignalError(w http.ResponseWriter, err error) error {
	if err == nil {
		err = errors.New("unknown signaling error")
	}
	return json.NewEncoder(w).Encode(&signalError{Error: err.Error()})
}
