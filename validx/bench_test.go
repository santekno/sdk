package validx_test

import (
	"testing"

	"github.com/santekno/sdk/validx"
)

func BenchmarkParseNIK(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = validx.ParseNIK("3201010101800001")
	}
}

func BenchmarkFormatIDR(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = validx.FormatIDR(1_500_000)
	}
}
