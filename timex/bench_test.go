package timex_test

import (
	"testing"
	"time"

	"github.com/santekno/sdk/timex"
)

func BenchmarkFormatID(b *testing.B) {
	t := time.Date(2026, 5, 17, 14, 30, 0, 0, timex.Jakarta)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = timex.FormatID(t, timex.LongID)
	}
}

func BenchmarkBusinessDaysBetween(b *testing.B) {
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, timex.Jakarta)
	to := time.Date(2026, 3, 31, 0, 0, 0, 0, timex.Jakarta)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = timex.BusinessDaysBetween(from, to)
	}
}
