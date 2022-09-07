package http3

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	kHttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/go-kratos/kratos/v2/transport/http/binding"
	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/http3"
	"github.com/stretchr/testify/assert"
	api "github.com/tx7do/kratos-transport/_example/api/protobuf"
	"math/rand"
	"net/http"
	"strconv"
	"testing"
)

func HygrothermographHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("HygrothermographHandler [%s] [%s] [%s]\n", r.Proto, r.Method, r.RequestURI)

	if r.Method == "POST" {
		var in api.Hygrothermograph
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			fmt.Printf("decode error: %s\n", err.Error())
		}
		fmt.Printf("Humidity: %s Temperature: %s \n", in.Humidity, in.Temperature)
	}

	var out api.Hygrothermograph
	out.Humidity = strconv.FormatInt(int64(rand.Intn(100)), 10)
	out.Temperature = strconv.FormatInt(int64(rand.Intn(100)), 10)
	_ = json.NewEncoder(w).Encode(&out)
}

func TestServer(t *testing.T) {
	ctx := context.Background()

	srv := NewServer(
		WithAddress(":8800"),
	)

	srv.HandleFunc("/hygrothermograph", HygrothermographHandler)

	if err := srv.Start(ctx); err != nil {
		panic(err)
	}

	defer func() {
		if err := srv.Stop(ctx); err != nil {
			t.Errorf("expected nil got %v", err)
		}
	}()
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

func CreateHygrothermograph(ctx context.Context, cli *kHttp.Client, in *api.Hygrothermograph, opts ...kHttp.CallOption) (*api.Hygrothermograph, error) {
	var out api.Hygrothermograph

	pattern := "/hygrothermograph"
	path := binding.EncodeURL(pattern, in, false)

	opts = append(opts, kHttp.Operation("/CreateHygrothermograph"))
	opts = append(opts, kHttp.PathTemplate(pattern))

	err := cli.Invoke(ctx, "POST", path, in, &out, opts...)
	if err != nil {
		return nil, err
	}

	return &out, nil
}

func TestClient(t *testing.T) {
	ctx := context.Background()

	var qconf quic.Config

	pool, err := x509.SystemCertPool()
	assert.Nil(t, err)

	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		RootCAs:            pool,
	}
	cli, err := kHttp.NewClient(ctx,
		kHttp.WithEndpoint("127.0.0.1:8800"),
		kHttp.WithTLSConfig(tlsConf),
		kHttp.WithTransport(&http3.RoundTripper{TLSClientConfig: tlsConf, QuicConfig: &qconf}),
	)
	assert.Nil(t, err)
	assert.NotNil(t, cli)

	var req api.Hygrothermograph
	req.Humidity = strconv.FormatInt(int64(rand.Intn(100)), 10)
	req.Temperature = strconv.FormatInt(int64(rand.Intn(100)), 10)

	resp, err := GetHygrothermograph(ctx, cli, &req, kHttp.EmptyCallOption{})
	assert.Nil(t, err)
	t.Log(resp)

	resp, err = CreateHygrothermograph(ctx, cli, &req, kHttp.EmptyCallOption{})
	assert.Nil(t, err)
	t.Log(resp)
}
