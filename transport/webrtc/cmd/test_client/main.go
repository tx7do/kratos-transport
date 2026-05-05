package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	webrtcTransport "github.com/tx7do/kratos-transport/transport/webrtc"
)

func main() {
	signalURL := flag.String("signal", "http://127.0.0.1:9999/signal", "signal server URL")
	authorization := flag.String("auth", "", "Authorization header value")
	label := flag.String("label", "kratos", "data channel label")
	mode := flag.String("mode", "binary", "payload mode: binary or text")
	msgType := flag.Uint("type", 1, "message type")
	payload := flag.String("payload", `{"type":1,"sender":"go","message":"hello from go client"}`, "payload JSON or plain text")
	keepalive := flag.Bool("keepalive", true, "wait for incoming messages until Ctrl+C")
	flag.Parse()

	var payloadType webrtcTransport.PayloadType = webrtcTransport.PayloadTypeBinary
	if *mode == "text" {
		payloadType = webrtcTransport.PayloadTypeText
	}

	client := webrtcTransport.NewClient(
		webrtcTransport.WithSignalURL(*signalURL),
		webrtcTransport.WithAuthorization(*authorization),
		webrtcTransport.WithClientPayloadType(payloadType),
		webrtcTransport.WithClientDataChannelLabel(*label),
		webrtcTransport.WithClientConnectTimeout(10*time.Second),
	)

	registerLogHandler(client, webrtcTransport.NetMessageType(*msgType))

	if err := client.Connect(); err != nil {
		fmt.Fprintf(os.Stderr, "connect failed: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = client.Disconnect() }()

	msg := buildMessage(*payload)
	if err := client.SendMessage(webrtcTransport.NetMessageType(*msgType), msg); err != nil {
		fmt.Fprintf(os.Stderr, "send failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("sent message type=%d\n", *msgType)

	if !*keepalive {
		return
	}

	fmt.Println("client connected, press Ctrl+C to quit...")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

}

func registerLogHandler(client *webrtcTransport.Client, messageType webrtcTransport.NetMessageType) {
	client.RegisterMessageHandler(messageType, func(payload webrtcTransport.MessagePayload) error {
		switch v := payload.(type) {
		case []byte:
			fmt.Printf("recv type=%d payload=%s\n", messageType, string(v))
		default:
			fmt.Printf("recv type=%d payload=%v\n", messageType, v)
		}
		return nil
	}, nil)
}

func buildMessage(raw string) any {
	if raw == "" {
		return map[string]any{}
	}

	var obj any
	if err := json.Unmarshal([]byte(raw), &obj); err == nil {
		return obj
	}

	if n, err := strconv.Atoi(raw); err == nil {
		return n
	}

	return raw
}
