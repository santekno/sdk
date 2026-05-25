package convx_test

import (
	"errors"
	"testing"

	"github.com/santekno/sdk/convx"
)

func TestToInt(t *testing.T) {
	cases := []struct {
		in   any
		want int
		ok   bool
	}{
		{42, 42, true},
		{"99", 99, true},
		{"bad", 0, false},
		{3.7, 3, true},
		{true, 1, true},
		{false, 0, true},
		{nil, 0, false},
	}
	for _, c := range cases {
		got, ok := convx.ToInt(c.in)
		if ok != c.ok || got != c.want {
			t.Errorf("ToInt(%v) = (%d, %v), want (%d, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestToString(t *testing.T) {
	if s := convx.ToString(42); s != "42" {
		t.Errorf("ToString(42) = %q, want %q", s, "42")
	}
	if s := convx.ToString("hi"); s != "hi" {
		t.Errorf("ToString(hi) = %q, want %q", s, "hi")
	}
	if s := convx.ToString(nil); s != "" {
		t.Errorf("ToString(nil) = %q, want empty", s)
	}
}

func TestToBool(t *testing.T) {
	if v, ok := convx.ToBool(true); !ok || !v {
		t.Error("ToBool(true) failed")
	}
	if v, ok := convx.ToBool("true"); !ok || !v {
		t.Error("ToBool(\"true\") failed")
	}
	if v, ok := convx.ToBool("bad"); ok || v {
		t.Error("ToBool(\"bad\") should fail")
	}
}

func TestToFloat64(t *testing.T) {
	if v, ok := convx.ToFloat64(3.14); !ok || v != 3.14 {
		t.Errorf("ToFloat64(3.14) = (%v, %v)", v, ok)
	}
	if v, ok := convx.ToFloat64("2.71"); !ok || v != 2.71 {
		t.Errorf("ToFloat64(\"2.71\") = (%v, %v)", v, ok)
	}
	if _, ok := convx.ToFloat64("nan!"); ok {
		t.Error("ToFloat64(\"nan!\") should fail")
	}
}

func TestMust(t *testing.T) {
	v := convx.Must(42, nil)
	if v != 42 {
		t.Errorf("Must(42, nil) = %d, want 42", v)
	}
	defer func() {
		if r := recover(); r == nil {
			t.Error("Must should panic on non-nil error")
		}
	}()
	_ = convx.Must(0, errors.New("boom"))
}
