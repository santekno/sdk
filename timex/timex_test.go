package timex_test

import (
	"strings"
	"testing"
	"time"

	"github.com/santekno/sdk/timex"
)

func TestFormatID_LongID(t *testing.T) {
	tm := time.Date(2026, 5, 18, 0, 0, 0, 0, timex.Jakarta)
	got := timex.FormatID(tm, timex.LongID)
	if !strings.Contains(got, "Senin") {
		t.Errorf("FormatID = %q, expected to contain Senin", got)
	}
	if !strings.Contains(got, "Mei") {
		t.Errorf("FormatID = %q, expected to contain Mei", got)
	}
	if !strings.Contains(got, "2026") {
		t.Errorf("FormatID = %q, expected to contain 2026", got)
	}
}

func TestFormatID_Zero(t *testing.T) {
	if got := timex.FormatID(time.Time{}, timex.LongID); got != "" {
		t.Errorf("FormatID(zero) = %q, want empty", got)
	}
}

func TestParseID_Roundtrip(t *testing.T) {
	tm := time.Date(2026, 5, 18, 0, 0, 0, 0, timex.Jakarta)
	s := timex.FormatID(tm, timex.LongID)
	back, err := timex.ParseID(timex.LongID, s)
	if err != nil {
		t.Fatalf("ParseID error: %v", err)
	}
	if back.Year() != 2026 || back.Month() != 5 || back.Day() != 18 {
		t.Errorf("ParseID roundtrip lost data: %v", back)
	}
}

func TestParseID_Invalid(t *testing.T) {
	if _, err := timex.ParseID(timex.LongID, "bogus value"); err == nil {
		t.Error("expected error for invalid input")
	}
}

func TestTimezones(t *testing.T) {
	if timex.Jakarta == nil || timex.Makassar == nil || timex.Jayapura == nil {
		t.Fatal("timezones not loaded")
	}
	_, offset := time.Now().In(timex.Jakarta).Zone()
	if offset != 7*3600 {
		t.Errorf("Jakarta offset = %d, want %d", offset, 7*3600)
	}
}

func TestToJakarta(t *testing.T) {
	utc := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	j := timex.ToJakarta(utc)
	if j.Location().String() == "UTC" {
		t.Errorf("ToJakarta did not change location")
	}
}

func TestAge(t *testing.T) {
	now := time.Now().In(timex.Jakarta)
	birth := time.Date(now.Year()-25, now.Month(), now.Day(), 0, 0, 0, 0, timex.Jakarta)
	if got := timex.Age(birth); got != 25 {
		t.Errorf("Age = %d, want 25", got)
	}
	future := time.Date(now.Year()+5, 1, 1, 0, 0, 0, 0, timex.Jakarta)
	if got := timex.Age(future); got != 0 {
		t.Errorf("Age(future) = %d, want 0", got)
	}
}

func TestIsIndonesianHoliday(t *testing.T) {
	indep := time.Date(2026, 8, 17, 0, 0, 0, 0, timex.Jakarta)
	if !timex.IsIndonesianHoliday(indep) {
		t.Error("17 August 2026 should be a holiday")
	}
	regular := time.Date(2026, 7, 7, 0, 0, 0, 0, timex.Jakarta)
	if timex.IsIndonesianHoliday(regular) {
		t.Error("7 July 2026 should not be a holiday")
	}
}

func TestIsBusinessDay(t *testing.T) {
	// 2026-05-18 is a Monday — business day
	mon := time.Date(2026, 5, 18, 0, 0, 0, 0, timex.Jakarta)
	if !timex.IsBusinessDay(mon) {
		t.Error("Monday should be business day")
	}
	// 2026-05-17 is a Sunday — not a business day
	sun := time.Date(2026, 5, 17, 0, 0, 0, 0, timex.Jakarta)
	if timex.IsBusinessDay(sun) {
		t.Error("Sunday should not be business day")
	}
	// 2026-08-17 (Independence Day) — holiday, not a business day even though Monday
	indep := time.Date(2026, 8, 17, 0, 0, 0, 0, timex.Jakarta)
	if timex.IsBusinessDay(indep) {
		t.Error("Independence Day should not be business day")
	}
}

func TestBusinessDaysBetween(t *testing.T) {
	// 2026-05-15 (Fri) → 2026-05-22 (Fri): 5 business days (Fri,Mon,Tue,Wed,Thu)
	start := time.Date(2026, 5, 15, 0, 0, 0, 0, timex.Jakarta)
	end := time.Date(2026, 5, 22, 0, 0, 0, 0, timex.Jakarta)
	if got := timex.BusinessDaysBetween(start, end); got != 5 {
		t.Errorf("BusinessDaysBetween = %d, want 5", got)
	}
	// end before start
	if got := timex.BusinessDaysBetween(end, start); got != 0 {
		t.Errorf("BusinessDaysBetween reversed = %d, want 0", got)
	}
}

func TestAddBusinessDays(t *testing.T) {
	mon := time.Date(2026, 5, 18, 0, 0, 0, 0, timex.Jakarta)
	got := timex.AddBusinessDays(mon, 5)
	// Mon + 5 business days = next Mon
	if got.Weekday() != time.Monday || got.Day() != 25 {
		t.Errorf("AddBusinessDays = %v, want next Monday", got)
	}
}

func TestIndonesianHolidays(t *testing.T) {
	hs := timex.IndonesianHolidays(2026)
	if len(hs) == 0 {
		t.Error("expected holidays for 2026")
	}
	if got := timex.IndonesianHolidays(1900); len(got) != 0 {
		t.Errorf("unknown year should return empty, got %d", len(got))
	}
}

func ExampleFormatID() {
	t := time.Date(2026, 5, 18, 0, 0, 0, 0, timex.Jakarta)
	_ = timex.FormatID(t, timex.LongID) // "Senin, 18 Mei 2026"
}
