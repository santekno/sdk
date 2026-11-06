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

func BenchmarkNewUmmAlQura(b *testing.B) {
	opt := timex.WithHijriMonthOffsets(timex.HijriMonthOffset{Year: 1446, Month: 10, Days: 1})
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := timex.NewUmmAlQura(opt); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFromCivilDate(b *testing.B) {
	cal, err := timex.NewUmmAlQura()
	if err != nil {
		b.Fatal(err)
	}
	t := time.Date(2026, 10, 28, 19, 0, 0, 0, timex.Jakarta)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := cal.FromCivilDate(t, timex.Jakarta); err != nil {
			b.Fatal(err)
		}
	}
}
