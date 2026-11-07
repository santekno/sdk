package hashx_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/santekno/sdk/hashx"
)

// RFC 2202 section 3 test cases for HMAC-SHA-1.
func TestHMACSHA1RFC2202(t *testing.T) {
	rep := func(b byte, n int) []byte { return bytes.Repeat([]byte{b}, n) }
	seq := make([]byte, 25)
	for i := range seq {
		seq[i] = byte(i + 1)
	}
	tests := []struct {
		name      string
		key, data []byte
		want      string
	}{
		{"1", rep(0x0b, 20), []byte("Hi There"), "b617318655057264e28bc0b6fb378c8ef146be00"},
		{"2", []byte("Jefe"), []byte("what do ya want for nothing?"), "effcdf6ae5eb2fa2d27416d5f184df9c259a7c79"},
		{"3", rep(0xaa, 20), rep(0xdd, 50), "125d7342b9ac11cd91a39af48aa17b4f63f175d3"},
		{"4", seq, rep(0xcd, 50), "4c9007f4026250c6bc8414f9bf50c86c2d7235da"},
		{"5", rep(0x0c, 20), []byte("Test With Truncation"), "4c1a03424b55e07fe7f27be1d58bb9324a9a5a04"},
		{"6 long key", rep(0xaa, 80), []byte("Test Using Larger Than Block-Size Key - Hash Key First"), "aa4ae5e15272d00e95705637ce8a3b55ed402112"},
		{"7 long key and data", rep(0xaa, 80), []byte("Test Using Larger Than Block-Size Key and Larger Than One Block-Size Data"), "e8e99d0f45237d786d6bbaa7965c7808bbff1a91"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hashx.HMACSHA1(tt.key, tt.data)
			if hex.EncodeToString(got) != tt.want {
				t.Errorf("HMACSHA1 = %x, want %s", got, tt.want)
			}
			if !hashx.ConstantTimeEqual(got, mustHex(t, tt.want)) {
				t.Error("ConstantTimeEqual must accept the expected MAC")
			}
		})
	}
}

func TestHMACSHA1Mismatch(t *testing.T) {
	a := hashx.HMACSHA1([]byte("k"), []byte("d"))
	if len(a) != 20 {
		t.Fatalf("length = %d, want 20", len(a))
	}
	if hashx.ConstantTimeEqual(a, hashx.HMACSHA1([]byte("k2"), []byte("d"))) {
		t.Error("different keys must differ")
	}
	if hashx.ConstantTimeEqual(a, hashx.HMACSHA1([]byte("k"), []byte("d2"))) {
		t.Error("different data must differ")
	}
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
