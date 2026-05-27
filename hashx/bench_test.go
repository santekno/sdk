package hashx_test

import (
	"testing"

	"github.com/santekno/sdk/hashx"
)

func BenchmarkHashPassword(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_, _ = hashx.HashPassword("benchmark-password-123")
	}
}
