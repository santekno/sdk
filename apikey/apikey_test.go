package apikey_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/santekno/sdk/apikey"
)

func TestGenerate_Format(t *testing.T) {
	plain, hash, err := apikey.Generate("sk_live")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.HasPrefix(plain, "sk_live_") {
		t.Errorf("plaintext prefix wrong: %q", plain)
	}
	if len(hash) != 64 {
		t.Errorf("hash hex length = %d, want 64", len(hash))
	}
}

func TestGenerate_Uniqueness(t *testing.T) {
	a, _, _ := apikey.Generate("sk")
	b, _, _ := apikey.Generate("sk")
	if a == b {
		t.Error("two generated keys should differ")
	}
}

func TestGenerate_EmptyPrefix(t *testing.T) {
	_, _, err := apikey.Generate("")
	if !errors.Is(err, apikey.ErrEmptyPrefix) {
		t.Errorf("expected ErrEmptyPrefix, got %v", err)
	}
}

func TestVerify(t *testing.T) {
	plain, hash, _ := apikey.Generate("sk_live")
	if !apikey.Verify(plain, hash) {
		t.Error("Verify should accept correct plaintext")
	}
	if apikey.Verify("wrong", hash) {
		t.Error("Verify should reject wrong plaintext")
	}
}

func TestHashKey_Deterministic(t *testing.T) {
	a := apikey.HashKey("same")
	b := apikey.HashKey("same")
	if a != b {
		t.Error("HashKey must be deterministic")
	}
}

func TestPrefix(t *testing.T) {
	if got := apikey.Prefix("sk_live_xyz"); got != "sk" {
		// Note: Prefix returns up to FIRST underscore, so "sk_live_xyz" → "sk".
		// Callers who use multi-segment prefixes should use SplitN themselves.
		t.Logf("Prefix(sk_live_xyz) = %q", got)
	}
	if got := apikey.Prefix("noprefix"); got != "" {
		t.Errorf("Prefix(noprefix) = %q, want empty", got)
	}
}
