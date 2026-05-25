package hashx_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/santekno/sdk/hashx"
)

func TestHashPassword_Roundtrip(t *testing.T) {
	hash, err := hashx.HashPassword("my-password")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Errorf("hash prefix unexpected: %q", hash)
	}
	if !hashx.VerifyPassword("my-password", hash) {
		t.Error("VerifyPassword should accept correct password")
	}
	if hashx.VerifyPassword("wrong", hash) {
		t.Error("VerifyPassword should reject wrong password")
	}
}

func TestHashPassword_Empty(t *testing.T) {
	_, err := hashx.HashPassword("")
	if !errors.Is(err, hashx.ErrEmptyPassword) {
		t.Errorf("expected ErrEmptyPassword, got %v", err)
	}
}

func TestVerifyPassword_BadFormat(t *testing.T) {
	if hashx.VerifyPassword("any", "not-a-real-hash") {
		t.Error("malformed hash should not verify")
	}
	if hashx.VerifyPassword("any", "$argon2id$x") {
		t.Error("partial hash should not verify")
	}
}

func TestBcrypt_Roundtrip(t *testing.T) {
	hash, err := hashx.HashPasswordBcrypt("pw", 4) // low cost for test speed
	if err != nil {
		t.Fatalf("HashPasswordBcrypt error: %v", err)
	}
	if !hashx.VerifyPasswordBcrypt("pw", hash) {
		t.Error("VerifyPasswordBcrypt should accept correct password")
	}
	if hashx.VerifyPasswordBcrypt("wrong", hash) {
		t.Error("VerifyPasswordBcrypt should reject wrong")
	}
}

func TestBcrypt_DefaultCost(t *testing.T) {
	hash, err := hashx.HashPasswordBcrypt("pw", 0)
	if err != nil {
		t.Fatalf("HashPasswordBcrypt error: %v", err)
	}
	if !hashx.VerifyPasswordBcrypt("pw", hash) {
		t.Error("verify failed with default cost")
	}
}

func TestBcrypt_Empty(t *testing.T) {
	_, err := hashx.HashPasswordBcrypt("", 4)
	if !errors.Is(err, hashx.ErrEmptyPassword) {
		t.Errorf("expected ErrEmptyPassword, got %v", err)
	}
}

func TestConstantTimeEqual(t *testing.T) {
	if !hashx.ConstantTimeEqual([]byte("abc"), []byte("abc")) {
		t.Error("equal byte slices should compare equal")
	}
	if hashx.ConstantTimeEqual([]byte("abc"), []byte("xyz")) {
		t.Error("unequal byte slices should not compare equal")
	}
}

func TestHMAC(t *testing.T) {
	key := []byte("secret")
	data := []byte("hello world")
	mac256 := hashx.HMACSHA256(key, data)
	mac512 := hashx.HMACSHA512(key, data)
	if len(mac256) != 32 {
		t.Errorf("HMAC-SHA256 length = %d, want 32", len(mac256))
	}
	if len(mac512) != 64 {
		t.Errorf("HMAC-SHA512 length = %d, want 64", len(mac512))
	}
	// Deterministic
	if !hashx.ConstantTimeEqual(mac256, hashx.HMACSHA256(key, data)) {
		t.Error("HMAC-SHA256 must be deterministic")
	}
}
