package thrift

import (
	"context"
	"testing"

	api "github.com/tx7do/kratos-transport/testing/api/thrift/gen-go/hygrothermograph"
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
}
