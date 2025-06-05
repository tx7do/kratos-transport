package keepalive

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeepAliveService(t *testing.T) {
	ctx := context.Background()

	svc := NewServer()
	assert.NotNil(t, svc)

	err := svc.Start(ctx)
	assert.Nil(t, err)
	defer svc.Stop(ctx)
}
