package ptrx_test

import (
	"testing"

	"github.com/santekno/sdk/ptrx"
)

func TestOf(t *testing.T) {
	p := ptrx.Of(42)
	if p == nil || *p != 42 {
		t.Fatalf("Of(42) = %v, want pointer to 42", p)
	}
	s := ptrx.Of("hello")
	if s == nil || *s != "hello" {
		t.Fatalf("Of(hello) = %v, want pointer to hello", s)
	}
}

func TestDeref(t *testing.T) {
	v := 99
	if got := ptrx.Deref(&v, 0); got != 99 {
		t.Errorf("Deref(&99, 0) = %d, want 99", got)
	}
	if got := ptrx.Deref((*int)(nil), 7); got != 7 {
		t.Errorf("Deref(nil, 7) = %d, want 7", got)
	}
}

func TestIsNil(t *testing.T) {
	if ptrx.IsNil((*int)(nil)) != true {
		t.Error("IsNil(nil) should be true")
	}
	v := 1
	if ptrx.IsNil(&v) != false {
		t.Error("IsNil(&v) should be false")
	}
}

func TestEqual(t *testing.T) {
	a, b := 5, 5
	if !ptrx.Equal(&a, &b) {
		t.Error("Equal(&5, &5) should be true")
	}
	c := 6
	if ptrx.Equal(&a, &c) {
		t.Error("Equal(&5, &6) should be false")
	}
	if !ptrx.Equal((*int)(nil), (*int)(nil)) {
		t.Error("Equal(nil, nil) should be true")
	}
	if ptrx.Equal(&a, (*int)(nil)) {
		t.Error("Equal(&5, nil) should be false")
	}
}

func ExampleOf() {
	p := ptrx.Of(42)
	_ = p // *int pointing to 42
}

func ExampleDeref() {
	var p *int
	v := ptrx.Deref(p, 0) // v == 0
	_ = v
}
