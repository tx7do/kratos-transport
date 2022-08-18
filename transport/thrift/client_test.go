package thrift

import (
	"context"
	"github.com/tx7do/kratos-transport/_example/api/thrift/gen-go/api"
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	conn, err := Dial(
		WithEndpoint("localhost:7700"),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	client := api.NewHygrothermographServiceClient(conn.Client)

	reply, err := client.GetHygrothermograph(context.Background())
	//t.Log(err)
	if err != nil {
		t.Errorf("failed to call: %v", err)
	}
	t.Log(*reply.Humidity, *reply.Temperature)

	time.Sleep(time.Second)
}
