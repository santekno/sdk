package mapx_test

import (
	"sort"
	"testing"

	"github.com/santekno/sdk/mapx"
)

func TestKeysValues(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	keys := mapx.Keys(m)
	sort.Strings(keys)
	if len(keys) != 3 || keys[0] != "a" {
		t.Errorf("Keys = %v", keys)
	}
	vs := mapx.Values(m)
	if len(vs) != 3 {
		t.Errorf("Values len = %d", len(vs))
	}
}

func TestEntriesFromEntries(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2}
	es := mapx.Entries(m)
	back := mapx.FromEntries(es)
	if back["a"] != 1 || back["b"] != 2 {
		t.Errorf("roundtrip failed: %v", back)
	}
}

func TestInvert(t *testing.T) {
	inv := mapx.Invert(map[string]int{"a": 1, "b": 2})
	if inv[1] != "a" || inv[2] != "b" {
		t.Errorf("Invert = %v", inv)
	}
}

func TestMapKeys(t *testing.T) {
	m := mapx.MapKeys(map[string]int{"a": 1}, func(k string) string { return k + "!" })
	if m["a!"] != 1 {
		t.Errorf("MapKeys = %v", m)
	}
}

func TestMapValues(t *testing.T) {
	m := mapx.MapValues(map[string]int{"a": 1, "b": 2}, func(v int) int { return v * 10 })
	if m["a"] != 10 || m["b"] != 20 {
		t.Errorf("MapValues = %v", m)
	}
}

func TestFilter(t *testing.T) {
	m := mapx.Filter(map[string]int{"a": 1, "b": 2, "c": 3}, func(_ string, v int) bool { return v >= 2 })
	if len(m) != 2 {
		t.Errorf("Filter = %v", m)
	}
}

func TestMerge(t *testing.T) {
	a := map[string]int{"a": 1, "b": 2}
	b := map[string]int{"b": 20, "c": 3}
	m := mapx.Merge(a, b)
	if m["a"] != 1 || m["b"] != 20 || m["c"] != 3 {
		t.Errorf("Merge = %v", m)
	}
}
