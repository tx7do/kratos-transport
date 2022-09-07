package webtransport

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	api "github.com/tx7do/kratos-transport/_example/api/manual"
	"os"
	"os/signal"
	"syscall"
	"testing"
)

var testServer *Server

func TestServer(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	// https://localhost:8800/webtransport
	srv := NewServer(
		WithAddress(":8800"),
		WithPath("/webtransport"),
		WithConnectHandle(handleConnect),
		WithTLSConfig(NewTlsConfig("./cert/server.key", "./cert/server.crt", "")),
		//WithCodec(encoding.GetCodec("json")),
	)

	testServer = srv

	if err := srv.Start(ctx); err != nil {
		panic(err)
	}

	defer func() {
		if err := srv.Stop(ctx); err != nil {
			t.Errorf("expected nil got %v", err)
		}
	}()

	<-interrupt
}

func handleConnect(sessionId SessionID, register bool) {
	if register {
		log.Infof("%d registered\n", sessionId)
	} else {
		log.Infof("%d unregistered\n", sessionId)
	}
}

func handleChatMessage(sessionId SessionID, message *api.ChatMessage) error {
	log.Infof("[%d] Payload: %v\n", sessionId, message)

	return nil
}

func TestClient(t *testing.T) {
	client := NewClient(
		//WithClientTLSConfig(NewTlsConfig("", "", "./cert/ca.crt")),
		WithServerUrl("https://localhost:8800/webtransport"),
	)
	err := client.Start()
	assert.Nil(t, err)
}
