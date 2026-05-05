package webrtc

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

const messageTypeChat NetMessageType = 1

type chatMessage struct {
	Type    int    `json:"type"`
	Sender  string `json:"sender"`
	Message string `json:"message"`
}

func waitUntil(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("condition not met within %s", timeout)
}

func startServerAsync(t *testing.T, srv *Server) chan error {
	t.Helper()
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(t.Context())
	}()

	waitUntil(t, 2*time.Second, func() bool {
		return srv.lis != nil && srv.running
	})

	return errCh
}

func TestServer(t *testing.T) {
	//t.Skip("manual smoke test: starts a fixed-port server and waits for OS signal")

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	srv := NewServer(
		WithAddress("127.0.0.1:9999"),
		WithPath("/signal"),
		WithCodec("json"),
	)

	RegisterServerMessageHandler(srv, messageTypeChat, func(_ SessionID, msg *chatMessage) error {
		t.Logf("Received chat message: sender=%s message=%s", msg.Sender, msg.Message)
		return nil
	})

	_ = startServerAsync(t, srv)

	ep, err := srv.Endpoint()
	if err != nil {
		t.Fatalf("Endpoint() error = %v", err)
	}
	if ep == nil || ep.Path != "/signal" {
		t.Fatalf("unexpected endpoint: %#v", ep)
	}

	defer srv.Stop(t.Context())

	<-interrupt
}

func TestServerStartStopAndEndpoint(t *testing.T) {
	srv := NewServer(
		WithAddress("127.0.0.1:0"),
		WithPath("/signal"),
		WithCodec("json"),
	)

	RegisterServerMessageHandler(srv, messageTypeChat, func(_ SessionID, msg *chatMessage) error {
		t.Logf("Received chat message: sender=%s message=%s", msg.Sender, msg.Message)
		return nil
	})

	errCh := startServerAsync(t, srv)

	ep, err := srv.Endpoint()
	if err != nil {
		t.Fatalf("Endpoint() error = %v", err)
	}
	if ep == nil || ep.Path != "/signal" {
		t.Fatalf("unexpected endpoint: %#v", ep)
	}

	if err = srv.Stop(t.Context()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if err = srv.Stop(t.Context()); err != nil {
		t.Fatalf("second Stop() should be idempotent, got %v", err)
	}

	select {
	case runErr := <-errCh:
		if runErr != nil {
			t.Fatalf("Start() returned error: %v", runErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server goroutine did not exit")
	}
}

func TestSignalHandler_MethodNotAllowed(t *testing.T) {
	srv := NewServer(
		WithAddress("127.0.0.1:0"),
		WithPath("/signal"),
	)

	errCh := startServerAsync(t, srv)
	defer func() {
		_ = srv.Stop(t.Context())
		<-errCh
	}()

	resp, err := http.Get("http://" + srv.lis.Addr().String() + "/signal")
	if err != nil {
		t.Fatalf("GET /signal failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", resp.StatusCode)
	}
}

func TestSignalHandler_CheckOriginDenied(t *testing.T) {
	srv := NewServer(
		WithAddress("127.0.0.1:0"),
		WithPath("/signal"),
		WithCheckOriginFunc(func(_ *http.Request) bool { return false }),
	)

	errCh := startServerAsync(t, srv)
	defer func() {
		_ = srv.Stop(t.Context())
		<-errCh
	}()

	req, err := http.NewRequest(http.MethodPost, "http://"+srv.lis.Addr().String()+"/signal", bytes.NewBufferString(`{"offer":{}}`))
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	req.Header.Set("Origin", "https://blocked.example")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /signal failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

func TestSignalHandler_BadRequestOnInvalidOffer(t *testing.T) {
	srv := NewServer(
		WithAddress("127.0.0.1:0"),
		WithPath("/signal"),
	)

	errCh := startServerAsync(t, srv)
	defer func() {
		_ = srv.Stop(t.Context())
		<-errCh
	}()

	req, err := http.NewRequest(http.MethodPost, "http://"+srv.lis.Addr().String()+"/signal", bytes.NewBufferString(`{"not_offer":true}`))
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	req.Header.Set("Origin", "https://allowed.example")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /signal failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") == "" {
		t.Fatal("expected Access-Control-Allow-Origin header")
	}

	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		t.Fatal("expected non-empty error response body")
	}
}

func TestSignalHandler_CORSPreflight(t *testing.T) {
	srv := NewServer(
		WithAddress("127.0.0.1:0"),
		WithPath("/signal"),
	)

	errCh := startServerAsync(t, srv)
	defer func() {
		_ = srv.Stop(t.Context())
		<-errCh
	}()

	req, err := http.NewRequest(http.MethodOptions, "http://"+srv.lis.Addr().String()+"/signal", nil)
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	req.Header.Set("Origin", "null")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("OPTIONS /signal failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") == "" {
		t.Fatal("expected Access-Control-Allow-Origin header")
	}
}

func TestBinaryPacketMarshalRoundtrip(t *testing.T) {
	msg := BinaryNetPacket{Type: 10000, Payload: []byte("Hello World")}

	buf, err := msg.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got BinaryNetPacket
	if err = got.Unmarshal(buf); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if got.Type != msg.Type {
		t.Fatalf("unexpected type: got %d want %d", got.Type, msg.Type)
	}
	if string(got.Payload) != string(msg.Payload) {
		t.Fatalf("unexpected payload: got %q want %q", string(got.Payload), string(msg.Payload))
	}
}
