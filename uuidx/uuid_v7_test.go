package uuidx

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestNewV7_Format(t *testing.T) {
	for i := 0; i < 1000; i++ {
		id := NewV7()
		if !IsValidUUID(id) {
			t.Fatalf("NewV7() = %q, not a valid UUID", id)
		}
		if id[14] != '7' {
			t.Fatalf("NewV7() = %q, version nibble %q, want 7", id, id[14])
		}
		if v := id[19]; v != '8' && v != '9' && v != 'a' && v != 'b' {
			t.Fatalf("NewV7() = %q, variant nibble %q, want 8-b", id, v)
		}
	}
}

func TestNewV7_EncodesTimestamp(t *testing.T) {
	now := time.UnixMilli(1792588406028) // 2026-10-21T13:13:26.028Z
	id := newV7(now)
	hexTS := strings.ReplaceAll(id[:13], "-", "")
	ms, err := strconv.ParseInt(hexTS, 16, 64)
	if err != nil {
		t.Fatal(err)
	}
	if ms != now.UnixMilli() {
		t.Fatalf("timestamp = %d, want %d", ms, now.UnixMilli())
	}
}

func TestNewV7_SortsByTime(t *testing.T) {
	base := time.UnixMilli(1792588406028)
	prev := newV7(base)
	for i := 1; i <= 100; i++ {
		next := newV7(base.Add(time.Duration(i) * time.Millisecond))
		if next <= prev {
			t.Fatalf("id at +%dms %q does not sort after %q", i, next, prev)
		}
		prev = next
	}
}

func TestNewV7_Unique(t *testing.T) {
	seen := make(map[string]bool, 10000)
	for i := 0; i < 10000; i++ {
		id := NewV7()
		if seen[id] {
			t.Fatalf("duplicate id %q", id)
		}
		seen[id] = true
	}
}
