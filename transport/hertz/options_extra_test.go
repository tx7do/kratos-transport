package hertz

import (
	"testing"
	"time"

	hertz "github.com/cloudwego/hertz/pkg/app/server"

	"github.com/stretchr/testify/assert"
)

func TestServerOptions(t *testing.T) {
	srv := NewServer(
		WithAddress("127.0.0.1:8801"),
		WithServerOptions(
			hertz.WithMaxRequestBodySize(16<<20),
		),
	)

	assert.Equal(t, 16<<20, srv.GetOptions().MaxRequestBodySize)
}

func TestServerOptionsAppend(t *testing.T) {
	srv := NewServer(
		WithAddress("127.0.0.1:8801"),
		WithServerOptions(hertz.WithMaxRequestBodySize(16<<20)),
		WithServerOptions(hertz.WithReadTimeout(time.Second)),
	)

	assert.Equal(t, 16<<20, srv.GetOptions().MaxRequestBodySize)
	assert.Equal(t, time.Second, srv.GetOptions().ReadTimeout)
}

func TestWithStrictSlash(t *testing.T) {
	// hertz 默认 RedirectTrailingSlash=true，WithStrictSlash(false) 应能关闭。
	on := NewServer(
		WithAddress("127.0.0.1:8801"),
		WithStrictSlash(true),
	)
	assert.True(t, on.GetOptions().RedirectTrailingSlash)

	off := NewServer(
		WithAddress("127.0.0.1:8801"),
		WithStrictSlash(false),
	)
	assert.False(t, off.GetOptions().RedirectTrailingSlash)

	// WithStrictSlash 与 WithServerOptions 依次追加，互不覆盖。
	both := NewServer(
		WithAddress("127.0.0.1:8801"),
		WithStrictSlash(false),
		WithServerOptions(hertz.WithMaxRequestBodySize(16<<20)),
	)
	assert.False(t, both.GetOptions().RedirectTrailingSlash)
	assert.Equal(t, 16<<20, both.GetOptions().MaxRequestBodySize)
}
