package cryptox_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/santekno/sdk/cryptox"
)

func TestAESGCM_Roundtrip(t *testing.T) {
	key, err := cryptox.GenerateAESKey()
	if err != nil {
		t.Fatalf("GenerateAESKey: %v", err)
	}
	plaintext := []byte("hello, dunia")
	ct, err := cryptox.EncryptAESGCM(key, plaintext)
	if err != nil {
		t.Fatalf("EncryptAESGCM: %v", err)
	}
	pt, err := cryptox.DecryptAESGCM(key, ct)
	if err != nil {
		t.Fatalf("DecryptAESGCM: %v", err)
	}
	if !bytes.Equal(pt, plaintext) {
		t.Errorf("roundtrip mismatch: got %q, want %q", pt, plaintext)
	}
}

func TestAESGCM_InvalidKey(t *testing.T) {
	_, err := cryptox.EncryptAESGCM([]byte("short"), []byte("data"))
	if !errors.Is(err, cryptox.ErrInvalidKey) {
		t.Errorf("expected ErrInvalidKey, got %v", err)
	}
	_, err = cryptox.DecryptAESGCM([]byte("short"), []byte("data"))
	if !errors.Is(err, cryptox.ErrInvalidKey) {
		t.Errorf("expected ErrInvalidKey, got %v", err)
	}
}

func TestAESGCM_Tampered(t *testing.T) {
	key, _ := cryptox.GenerateAESKey()
	ct, _ := cryptox.EncryptAESGCM(key, []byte("secret"))
	ct[len(ct)-1] ^= 0xff // flip a bit in the auth tag
	if _, err := cryptox.DecryptAESGCM(key, ct); err == nil {
		t.Error("expected decryption error for tampered ciphertext")
	}
}

func TestAESGCM_Short(t *testing.T) {
	key, _ := cryptox.GenerateAESKey()
	if _, err := cryptox.DecryptAESGCM(key, []byte("x")); err == nil {
		t.Error("expected error for too-short ciphertext")
	}
}

func TestChaCha20_Roundtrip(t *testing.T) {
	key := make([]byte, cryptox.ChaCha20KeyLen)
	for i := range key {
		key[i] = byte(i)
	}
	plaintext := []byte("hello world")
	ct, err := cryptox.EncryptChaCha20(key, plaintext)
	if err != nil {
		t.Fatalf("EncryptChaCha20: %v", err)
	}
	pt, err := cryptox.DecryptChaCha20(key, ct)
	if err != nil {
		t.Fatalf("DecryptChaCha20: %v", err)
	}
	if !bytes.Equal(pt, plaintext) {
		t.Errorf("roundtrip mismatch")
	}
}

func TestChaCha20_InvalidKey(t *testing.T) {
	if _, err := cryptox.EncryptChaCha20([]byte("short"), []byte("data")); !errors.Is(err, cryptox.ErrInvalidKey) {
		t.Error("expected ErrInvalidKey")
	}
	if _, err := cryptox.DecryptChaCha20([]byte("short"), []byte("data")); !errors.Is(err, cryptox.ErrInvalidKey) {
		t.Error("expected ErrInvalidKey")
	}
}

func TestChaCha20_Short(t *testing.T) {
	key := make([]byte, cryptox.ChaCha20KeyLen)
	if _, err := cryptox.DecryptChaCha20(key, []byte("x")); err == nil {
		t.Error("expected error for too-short ciphertext")
	}
}

func TestEd25519(t *testing.T) {
	pub, priv, err := cryptox.GenerateEd25519KeyPair()
	if err != nil {
		t.Fatalf("GenerateEd25519KeyPair: %v", err)
	}
	msg := []byte("important message")
	sig := cryptox.SignEd25519(priv, msg)
	if !cryptox.VerifyEd25519(pub, msg, sig) {
		t.Error("VerifyEd25519 should accept valid signature")
	}
	if cryptox.VerifyEd25519(pub, []byte("tampered"), sig) {
		t.Error("VerifyEd25519 should reject tampered message")
	}
}

func TestRSA_Roundtrip(t *testing.T) {
	priv, err := cryptox.GenerateRSAKeyPair(2048)
	if err != nil {
		t.Fatalf("GenerateRSAKeyPair: %v", err)
	}
	msg := []byte("hello")
	ct, err := cryptox.EncryptRSAOAEP(&priv.PublicKey, msg)
	if err != nil {
		t.Fatalf("EncryptRSAOAEP: %v", err)
	}
	pt, err := cryptox.DecryptRSAOAEP(priv, ct)
	if err != nil {
		t.Fatalf("DecryptRSAOAEP: %v", err)
	}
	if !bytes.Equal(pt, msg) {
		t.Error("RSA roundtrip mismatch")
	}
}
