package mqtt

import (
	"reflect"
	"testing"
)

func TestTransport_Kind(t *testing.T) {
	o := &Transport{}
	if !reflect.DeepEqual(KindMQTT, o.Kind()) {
		t.Errorf("expect %v, got %v", KindMQTT, o.Kind())
	}
}

func TestTransport_Endpoint(t *testing.T) {
	v := "hello"
	o := &Transport{endpoint: v}
	if !reflect.DeepEqual(v, o.Endpoint()) {
		t.Errorf("expect %v, got %v", v, o.Endpoint())
	}
}

func TestTransport_Operation(t *testing.T) {
	v := "hello"
	o := &Transport{operation: v}
	if !reflect.DeepEqual(v, o.Operation()) {
		t.Errorf("expect %v, got %v", v, o.Operation())
	}
}
