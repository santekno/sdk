package cryptox_test

import (
	"testing"

	"github.com/santekno/sdk/cryptox"
)

func BenchmarkEncryptAESGCM(b *testing.B) {
	key, _ := cryptox.GenerateAESKey()
	plaintext := make([]byte, 1024) // 1 KiB
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = cryptox.EncryptAESGCM(key, plaintext)
	}
}

func BenchmarkDecryptAESGCM(b *testing.B) {
	key, _ := cryptox.GenerateAESKey()
	plaintext := make([]byte, 1024)
	ciphertext, _ := cryptox.EncryptAESGCM(key, plaintext)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = cryptox.DecryptAESGCM(key, ciphertext)
	}
}
