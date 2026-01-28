package broker

import (
	"reflect"
	"testing"
)

func TestMessage_GetHeadersAnd_GetHeader(t *testing.T) {
	// nil headers
	var m Message
	if got := m.GetHeaders(); got != nil {
		t.Fatalf("GetHeaders() = %v, want nil", got)
	}
	if got := m.GetHeader("missing"); got != "" {
		t.Fatalf("GetHeader() with nil headers = %q, want empty", got)
	}

	// existing headers
	m.Headers = Headers{"k": "v"}
	if got := m.GetHeaders(); !reflect.DeepEqual(got, Headers{"k": "v"}) {
		t.Fatalf("GetHeaders() = %v, want %v", got, Headers{"k": "v"})
	}
	if got := m.GetHeader("k"); got != "v" {
		t.Fatalf("GetHeader(\"k\") = %q, want %q", got, "v")
	}
	if got := m.GetHeader("nope"); got != "" {
		t.Fatalf("GetHeader(missing) = %q, want empty", got)
	}
}

func TestMessage_HeadersCopy_Independence(t *testing.T) {
	orig := Headers{"a": "1", "b": "2"}
	m := Message{Headers: orig}

	cp := m.HeadersCopy()
	if !reflect.DeepEqual(cp, orig) {
		t.Fatalf("HeadersCopy() = %v, want %v", cp, orig)
	}

	// 修改原始 map 不应影响拷贝
	orig["a"] = "changed"
	if cp["a"] == "changed" {
		t.Fatalf("copy mutated after orig change: copy[a]=%q", cp["a"])
	}

	// 修改拷贝不应影响原始/消息中的 Headers
	cp["b"] = "changed2"
	if m.Headers["b"] == "changed2" {
		t.Fatalf("orig mutated after copy change: orig[b]=%q", m.Headers["b"])
	}

	// nil headers -> copy should be nil
	var m2 Message
	if got := m2.HeadersCopy(); got != nil {
		t.Fatalf("HeadersCopy() with nil headers = %v, want nil", got)
	}
}
