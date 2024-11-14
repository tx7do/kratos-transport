package keepalive

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestKeepAliveService(t *testing.T) {
	svc := NewKeepAliveService()
	assert.NotNil(t, svc)
	err := svc.Start()
	assert.Nil(t, err)
}
