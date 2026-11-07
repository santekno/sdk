package timex_test

import (
	"errors"
	"testing"
	"time"

	"github.com/santekno/sdk/timex"
)

func FuzzParseID(f *testing.F) {
	f.Add(timex.LongID, "Sabtu, 17 Mei 2026")
	f.Add(timex.DateID, "17-05-2026")
	f.Add(timex.ShortID, "17/05/26")
	f.Add("", "")
	f.Fuzz(func(t *testing.T, layout, value string) {
		_, _ = timex.ParseID(layout, value)
	})
}

func FuzzUmmAlQuraFromCivilDate(f *testing.F) {
	cal, err := timex.NewUmmAlQura()
	if err != nil {
		f.Fatal(err)
	}
	f.Add(int64(-1035158400), 0)   // first table day
	f.Add(int64(3404332799), 0)    // last table second
	f.Add(int64(1781542800), 7*60) // 1 Muharram 1448 H in WIB
	f.Add(int64(0), 14*60)
	f.Add(int64(-1<<62), -12*60)
	f.Add(int64(1<<62), 14*60)
	f.Fuzz(func(t *testing.T, sec int64, offsetMinutes int) {
		if offsetMinutes < -24*60 || offsetMinutes > 24*60 {
			t.Skip()
		}
		loc := time.FixedZone("fuzz", offsetMinutes*60)
		at := time.Unix(sec, 0)

		got, err := cal.FromCivilDate(at, loc)
		if err != nil {
			if !errors.Is(err, timex.ErrHijriOutOfRange) {
				t.Fatalf("unexpected error: %v", err)
			}
			return
		}
		if got.Year < 1356 || got.Year > 1500 || got.Month < 1 || got.Month > 12 {
			t.Fatalf("date outside the table: %+v", got)
		}
		if n := cal.MonthLength(got.Year, got.Month); got.Day < 1 || got.Day > n {
			t.Fatalf("day outside month of %d days: %+v", n, got)
		}

		// The result depends on the civil date only.
		y, m, d := at.In(loc).Date()
		same, err := cal.FromCivilDate(time.Date(y, m, d, 0, 0, 0, 0, time.UTC), time.UTC)
		if err != nil || same != got {
			t.Fatalf("civil date gave %+v, %v; want %+v", same, err, got)
		}
	})
}

func FuzzNewUmmAlQuraOffsets(f *testing.F) {
	f.Add(1447, 9, 1, 1447, 10, 1)
	f.Add(1448, 6, -1, 1448, 6, 1)
	f.Add(1356, 1, -1, 1500, 12, 1)
	f.Add(0, 0, 0, 0, 0, 0)
	f.Fuzz(func(t *testing.T, y1, m1, d1, y2, m2, d2 int) {
		cal, err := timex.NewUmmAlQura(timex.WithHijriMonthOffsets(
			timex.HijriMonthOffset{Year: y1, Month: m1, Days: d1},
			timex.HijriMonthOffset{Year: y2, Month: m2, Days: d2},
		))
		if err != nil {
			return
		}
		for year := 1356; year <= 1500; year++ {
			for month := 1; month <= 12; month++ {
				if n := cal.MonthLength(year, month); n != 29 && n != 30 {
					t.Fatalf("MonthLength(%d, %d) = %d", year, month, n)
				}
			}
		}
	})
}

func FuzzParseISODuration(f *testing.F) {
	for _, s := range []string{"PT1H2M3S", "P1DT2H", "P0D", "PT0.5S", "", "P", "PT", "P1Y", "pt1s", "PT-1S", "PT9223372036854775807S"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		d, err := timex.ParseISODuration(s)
		if err != nil {
			if !errors.Is(err, timex.ErrInvalidISODuration) || d != 0 {
				t.Fatalf("ParseISODuration(%q) = %v, %v", s, d, err)
			}
			return
		}
		if d < 0 {
			t.Fatalf("ParseISODuration(%q) = negative %v", s, d)
		}
	})
}
