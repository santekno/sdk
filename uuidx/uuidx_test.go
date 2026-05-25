package uuidx_test

import (
	"testing"

	"github.com/santekno/sdk/uuidx"
)

func TestNewUUID_Format(t *testing.T) {
	u := uuidx.NewUUID()
	if len(u) != 36 {
		t.Fatalf("UUID len = %d, want 36", len(u))
	}
	if !uuidx.IsValidUUID(u) {
		t.Errorf("Generated UUID %q is not valid", u)
	}
}

func TestNewUUID_Unique(t *testing.T) {
	a, b := uuidx.NewUUID(), uuidx.NewUUID()
	if a == b {
		t.Errorf("UUIDs should be unique, got %s twice", a)
	}
}

func TestIsValidUUID(t *testing.T) {
	cases := map[string]bool{
		"550e8400-e29b-41d4-a716-446655440000": true,
		"550e8400-e29b-41d4-a716-44665544000":  false, // too short
		"not-a-uuid":                           false,
		"550e8400e29b41d4a716446655440000":     false, // missing dashes
	}
	for in, want := range cases {
		if got := uuidx.IsValidUUID(in); got != want {
			t.Errorf("IsValidUUID(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestNewShortID(t *testing.T) {
	id := uuidx.NewShortID()
	if len(id) != 12 {
		t.Errorf("ShortID len = %d, want 12", len(id))
	}
}
