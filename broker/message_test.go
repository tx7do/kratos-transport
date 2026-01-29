package broker

import (
	"context"
	"reflect"
	"testing"
)

func TestCreator_Primitive(t *testing.T) {
	p := Creator[int]()
	if p == nil {
		t.Fatalf("Creator[int]() returned nil")
	}
	if *p != 0 {
		t.Fatalf("expected zero value, got %v", *p)
	}
	*p = 42
	if *p != 42 {
		t.Fatalf("assignment failed, got %v", *p)
	}
}

func TestCreator_Struct(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}

	p := Creator[Person]()
	if p == nil {
		t.Fatalf("Creator[Person]() returned nil")
	}
	if p.Name != "" || p.Age != 0 {
		t.Fatalf("expected zero-value Person, got %+v", *p)
	}
	p.Name = "Alice"
	p.Age = 30
	if p.Name != "Alice" || p.Age != 30 {
		t.Fatalf("assignment to created struct failed, got %+v", *p)
	}
}

func TestCreator_SliceAndMap(t *testing.T) {
	ps := Creator[[]int]()
	if ps == nil {
		t.Fatalf("Creator[[]int]() returned nil")
	}
	if *ps != nil {
		t.Fatalf("expected nil slice pointer value, got %v", *ps)
	}
	*ps = []int{1, 2, 3}
	if len(*ps) != 3 || (*ps)[0] != 1 {
		t.Fatalf("slice assignment failed, got %v", *ps)
	}

	pm := Creator[map[string]int]()
	if pm == nil {
		t.Fatalf("Creator[map[string]int]() returned nil")
	}
	if *pm != nil {
		t.Fatalf("expected nil map pointer value, got %v", *pm)
	}
	*pm = map[string]int{"a": 1}
	if (*pm)["a"] != 1 {
		t.Fatalf("map assignment failed, got %v", *pm)
	}
}

func TestCreator_UniqueInstances(t *testing.T) {
	a := Creator[int]()
	b := Creator[int]()
	if a == nil || b == nil {
		t.Fatalf("Creator returned nil pointer")
	}
	if a == b {
		t.Fatalf("expected distinct pointers on separate Creator calls, got same address")
	}
}

func TestMessage_SetWithAndGetHeader(t *testing.T) {
	var m Message
	m.SetHeader("a", "1")
	if got := m.GetHeader("a"); got != "1" {
		t.Fatalf("GetHeader = %q, want %q", got, "1")
	}

	// chaining
	m2 := (&Message{}).WithHeader("b", "2").WithHeader("c", "3")
	if m2.GetHeader("b") != "2" || m2.GetHeader("c") != "3" {
		t.Fatalf("WithHeader chaining failed: %#v", m2.Headers)
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

func TestMessage_MergeHeaders(t *testing.T) {
	m := &Message{}
	m.SetHeader("x", "1")
	m.MergeHeaders(Headers{"y": "2", "x": "3"})
	if got := m.GetHeader("x"); got != "3" {
		t.Fatalf("MergeHeaders overwrite failed: x=%q", got)
	}
	if got := m.GetHeader("y"); got != "2" {
		t.Fatalf("MergeHeaders add failed: y=%q", got)
	}
}

func TestMessage_MetadataCopyAndContext(t *testing.T) {
	m := &Message{}
	m.SetMetadata("m1", "v1")
	cp := m.MetadataCopy()
	if !reflect.DeepEqual(cp, Metadata{"m1": "v1"}) {
		t.Fatalf("MetadataCopy wrong: %v", cp)
	}

	// 修改原数据不影响拷贝
	m.Metadata["m1"] = "v2"
	if cp["m1"] == "v2" {
		t.Fatalf("metadata copy mutated after orig change")
	}

	// ExtractContext & GetMetadataFromContext
	ctx := context.Background()
	ctx2 := m.ExtractContext(ctx)
	if md := GetMetadataFromContext(ctx2); !reflect.DeepEqual(md, m.Metadata) {
		t.Fatalf("GetMetadataFromContext = %v, want %v", md, m.Metadata)
	}
}

func TestMessage_BodyBytes(t *testing.T) {
	var m Message
	if got := m.BodyBytes(); got != nil {
		t.Fatalf("BodyBytes with nil body = %v, want nil", got)
	}

	m.Body = []byte("abc")
	if got := m.BodyBytes(); string(got) != "abc" {
		t.Fatalf("BodyBytes returned %v, want %v", got, []byte("abc"))
	}

	m.Body = "string"
	if got := m.BodyBytes(); got != nil {
		t.Fatalf("BodyBytes with non-[]byte should be nil, got %v", got)
	}
}

func TestMessage_CloneShallowCopyHeadersMetadataAndBody(t *testing.T) {
	orig := &Message{
		ID:       "id1",
		Headers:  Headers{"h": "v"},
		Metadata: Metadata{"k": "v"},
		Body:     &struct{ A int }{A: 10},
		Key:      "key",
	}
	cl := orig.Clone()

	// headers / metadata should be deep-copied (maps), not pointer-equal
	if reflect.DeepEqual(cl.Headers, orig.Headers) == false {
		t.Fatalf("Clone headers content mismatch")
	}
	if &cl.Headers == &orig.Headers {
		t.Fatalf("Clone did not copy headers map (same address)")
	}
	if reflect.DeepEqual(cl.Metadata, orig.Metadata) == false {
		t.Fatalf("Clone metadata content mismatch")
	}
	if &cl.Metadata == &orig.Metadata {
		t.Fatalf("Clone did not copy metadata map (same address)")
	}

	// Body is shallow-copied (same pointer)
	if cl.Body == nil || orig.Body == nil {
		t.Fatalf("unexpected nil body")
	}
	if cl.Body != orig.Body {
		t.Fatalf("expected body to be the same pointer (shallow copy)")
	}
}

func TestMessage_InjectAndExtractToContext(t *testing.T) {
	type keyType string
	ctx := context.WithValue(context.Background(), keyType("user"), "alice")
	m := &Message{}
	m.InjectContext(ctx, keyType("user"))

	if got := m.GetHeader("user"); got != "alice" {
		t.Fatalf("InjectContext header missing: %q", got)
	}

	// ExtractToContext mapping header->ctx key
	ctx2 := m.ExtractToContext(context.Background(), map[string]any{"user": keyType("u")})
	if v := ctx2.Value(keyType("u")); v != "alice" {
		t.Fatalf("ExtractToContext failed, got %v", v)
	}
}

type ackable struct {
	called bool
}

// pointer receiver to mutate state
func (a *ackable) Ack() error {
	a.called = true
	return nil
}

func TestMessage_AckSuccess(t *testing.T) {
	a := &ackable{}
	m := &Message{Msg: a}
	if err := m.AckSuccess(); err != nil {
		t.Fatalf("AckSuccess returned error: %v", err)
	}
	if !a.called {
		t.Fatalf("Ack() was not called on underlying Msg")
	}

	// non-ackable Msg should not error
	m2 := &Message{Msg: struct{}{}}
	if err := m2.AckSuccess(); err != nil {
		t.Fatalf("AckSuccess on non-ackable should not error: %v", err)
	}
}
