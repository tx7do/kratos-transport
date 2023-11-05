package gozero

import (
	"context"
	"math/rand"
	"net/http"
	"strconv"
	"testing"

	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"

	kHttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/go-kratos/kratos/v2/transport/http/binding"
	"github.com/stretchr/testify/assert"

	api "github.com/tx7do/kratos-transport/testing/api/protobuf"
)

func TestServer(t *testing.T) {
	ctx := context.Background()

	srv := NewServer(
		WithAddress(":8800"),
	)

	//srv.Use(recover.New())
	//srv.Use(logger.New())

	type LoginRequest struct {
		User string `form:"user,options=a|b"`
	}

	srv.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/login",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			var req LoginRequest
			err := httpx.Parse(r, &req)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			httpx.OkJson(w, "hello, "+req.User)
		}})

	srv.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/hygrothermograph",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			var out api.Hygrothermograph
			out.Humidity = strconv.FormatInt(int64(rand.Intn(100)), 10)
			out.Temperature = strconv.FormatInt(int64(rand.Intn(100)), 10)
			httpx.OkJson(w, &out)
		}})

	if err := srv.Start(ctx); err != nil {
		panic(err)
	}

	defer func() {
		if err := srv.Stop(ctx); err != nil {
			t.Errorf("expected nil got %v", err)
		}
	}()
}

func TestClient(t *testing.T) {
	ctx := context.Background()

	cli, err := kHttp.NewClient(ctx,
		kHttp.WithEndpoint("127.0.0.1:8800"),
	)
	assert.Nil(t, err)
	assert.NotNil(t, cli)

	resp, err := GetHygrothermograph(ctx, cli, nil, kHttp.EmptyCallOption{})
	assert.Nil(t, err)
	t.Log(resp)
}

func GetHygrothermograph(ctx context.Context, cli *kHttp.Client, in *api.Hygrothermograph, opts ...kHttp.CallOption) (*api.Hygrothermograph, error) {
	var out api.Hygrothermograph

	pattern := "/hygrothermograph"
	path := binding.EncodeURL(pattern, in, true)

	opts = append(opts, kHttp.Operation("/GetHygrothermograph"))
	opts = append(opts, kHttp.PathTemplate(pattern))

	err := cli.Invoke(ctx, "GET", path, nil, &out, opts...)
	if err != nil {
		return nil, err
	}

	return &out, nil
}
