package timex_test

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/santekno/sdk/timex"
)

// hijriReferenceDates are Asia/Jakarta civil dates with their Umm al-Qura date.
// The values were sourced independently of any Intl or ICU output, from R.H. van
// Gent's Umm al-Qura tables, the AlAdhan API, and Kementerian Agama RI
// announcements, and are copied from the LAJ website
// (packages/core/src/hijri-reference.test.ts).
var hijriReferenceDates = []struct {
	date string
	want timex.HijriDate
}{
	{"2024-07-07", timex.HijriDate{Year: 1446, Month: 1, Day: 1}},
	{"2025-03-01", timex.HijriDate{Year: 1446, Month: 9, Day: 1}},
	{"2025-03-30", timex.HijriDate{Year: 1446, Month: 10, Day: 1}},
	{"2025-05-28", timex.HijriDate{Year: 1446, Month: 12, Day: 1}},
	{"2025-06-06", timex.HijriDate{Year: 1446, Month: 12, Day: 10}},
	{"2025-06-25", timex.HijriDate{Year: 1446, Month: 12, Day: 29}},
	{"2025-06-26", timex.HijriDate{Year: 1447, Month: 1, Day: 1}},
	{"2025-09-04", timex.HijriDate{Year: 1447, Month: 3, Day: 12}},
	{"2026-02-17", timex.HijriDate{Year: 1447, Month: 8, Day: 29}},
	{"2026-02-18", timex.HijriDate{Year: 1447, Month: 9, Day: 1}},
	{"2026-03-19", timex.HijriDate{Year: 1447, Month: 9, Day: 30}},
	{"2026-03-20", timex.HijriDate{Year: 1447, Month: 10, Day: 1}},
	{"2026-05-18", timex.HijriDate{Year: 1447, Month: 12, Day: 1}},
	{"2026-05-27", timex.HijriDate{Year: 1447, Month: 12, Day: 10}},
	{"2026-06-15", timex.HijriDate{Year: 1447, Month: 12, Day: 29}},
	{"2026-06-16", timex.HijriDate{Year: 1448, Month: 1, Day: 1}},
	{"2026-08-25", timex.HijriDate{Year: 1448, Month: 3, Day: 12}},
	{"2026-10-11", timex.HijriDate{Year: 1448, Month: 4, Day: 30}},
	{"2026-10-12", timex.HijriDate{Year: 1448, Month: 5, Day: 1}},
	{"2026-10-27", timex.HijriDate{Year: 1448, Month: 5, Day: 16}},
	{"2026-10-28", timex.HijriDate{Year: 1448, Month: 5, Day: 17}},
	{"2026-10-29", timex.HijriDate{Year: 1448, Month: 5, Day: 18}},
	{"2026-10-30", timex.HijriDate{Year: 1448, Month: 5, Day: 19}},
	{"2026-11-03", timex.HijriDate{Year: 1448, Month: 5, Day: 23}},
	{"2026-11-08", timex.HijriDate{Year: 1448, Month: 5, Day: 28}},
	{"2026-11-04", timex.HijriDate{Year: 1448, Month: 5, Day: 24}},
	{"2026-11-10", timex.HijriDate{Year: 1448, Month: 5, Day: 30}},
	{"2026-11-11", timex.HijriDate{Year: 1448, Month: 6, Day: 1}},
	{"2026-12-10", timex.HijriDate{Year: 1448, Month: 7, Day: 1}},
	{"2027-01-09", timex.HijriDate{Year: 1448, Month: 8, Day: 1}},
	{"2027-02-07", timex.HijriDate{Year: 1448, Month: 8, Day: 30}},
	{"2027-02-08", timex.HijriDate{Year: 1448, Month: 9, Day: 1}},
	{"2027-03-09", timex.HijriDate{Year: 1448, Month: 10, Day: 1}},
	{"2027-05-07", timex.HijriDate{Year: 1448, Month: 12, Day: 1}},
	{"2027-05-16", timex.HijriDate{Year: 1448, Month: 12, Day: 10}},
	{"2027-06-06", timex.HijriDate{Year: 1449, Month: 1, Day: 1}},
}

func newUmmAlQura(t *testing.T, opts ...timex.UmmAlQuraOption) *timex.UmmAlQura {
	t.Helper()
	cal, err := timex.NewUmmAlQura(opts...)
	if err != nil {
		t.Fatalf("NewUmmAlQura: %v", err)
	}
	return cal
}

func civilDate(t *testing.T, date string, loc *time.Location) time.Time {
	t.Helper()
	d, err := time.ParseInLocation(time.DateOnly, date, loc)
	if err != nil {
		t.Fatalf("parse %q: %v", date, err)
	}
	return d
}

func wantHijri(t *testing.T, cal *timex.UmmAlQura, in time.Time, loc *time.Location, want timex.HijriDate) {
	t.Helper()
	got, err := cal.FromCivilDate(in, loc)
	if err != nil {
		t.Fatalf("FromCivilDate(%s): %v", in.Format(time.RFC3339), err)
	}
	if got != want {
		t.Errorf("FromCivilDate(%s) = %+v, want %+v", in.Format(time.RFC3339), got, want)
	}
}

func TestUmmAlQura_ReferenceDates(t *testing.T) {
	cal := newUmmAlQura(t)
	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	if got := len(hijriReferenceDates); got != 36 {
		t.Fatalf("reference list has %d dates, want 36", got)
	}

	clock := []time.Duration{0, 18*time.Hour + 30*time.Minute, 23*time.Hour + 59*time.Minute}
	for _, tt := range hijriReferenceDates {
		t.Run(tt.date, func(t *testing.T) {
			midnight := civilDate(t, tt.date, timex.Jakarta)
			for _, d := range clock {
				at := midnight.Add(d)
				wantHijri(t, cal, at, timex.Jakarta, tt.want)
				// The input's own zone must not leak into the result.
				wantHijri(t, cal, at.In(time.UTC), timex.Jakarta, tt.want)
				wantHijri(t, cal, at.In(london), timex.Jakarta, tt.want)
			}
		})
	}
}

func TestUmmAlQura_InstantsUseCivilDateOfLocation(t *testing.T) {
	cal := newUmmAlQura(t)
	tests := []struct {
		instant string
		want    timex.HijriDate
	}{
		{"2026-10-28T22:00:00Z", timex.HijriDate{Year: 1448, Month: 5, Day: 18}}, // 29 Oct 05.00 WIB
		{"2026-10-28T11:00:00Z", timex.HijriDate{Year: 1448, Month: 5, Day: 17}},
		{"2026-10-30T04:45:00Z", timex.HijriDate{Year: 1448, Month: 5, Day: 19}},
		{"2026-11-03T12:30:00Z", timex.HijriDate{Year: 1448, Month: 5, Day: 23}},
		{"2026-11-08T01:00:00Z", timex.HijriDate{Year: 1448, Month: 5, Day: 28}},
	}
	for _, tt := range tests {
		t.Run(tt.instant, func(t *testing.T) {
			at, err := time.Parse(time.RFC3339, tt.instant)
			if err != nil {
				t.Fatal(err)
			}
			wantHijri(t, cal, at, timex.Jakarta, tt.want)
		})
	}
}

func TestUmmAlQura_DayChangesAtLocalMidnight(t *testing.T) {
	cal := newUmmAlQura(t)
	tests := []struct {
		name          string
		last          string // 23:59 WIB on the last day of a month, in UTC
		first         string // 00:00 WIB on the next day, in UTC
		before, after timex.HijriDate
	}{
		{"new year 1448", "2026-06-15T16:59:00Z", "2026-06-15T17:00:00Z",
			timex.HijriDate{Year: 1447, Month: 12, Day: 29}, timex.HijriDate{Year: 1448, Month: 1, Day: 1}},
		{"Ramadhan 1447", "2026-02-17T16:59:00Z", "2026-02-17T17:00:00Z",
			timex.HijriDate{Year: 1447, Month: 8, Day: 29}, timex.HijriDate{Year: 1447, Month: 9, Day: 1}},
		{"Syawal 1447", "2026-03-19T16:59:00Z", "2026-03-19T17:00:00Z",
			timex.HijriDate{Year: 1447, Month: 9, Day: 30}, timex.HijriDate{Year: 1447, Month: 10, Day: 1}},
		{"new year 1447", "2025-06-25T16:59:00Z", "2025-06-25T17:00:00Z",
			timex.HijriDate{Year: 1446, Month: 12, Day: 29}, timex.HijriDate{Year: 1447, Month: 1, Day: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before, err := time.Parse(time.RFC3339, tt.last)
			if err != nil {
				t.Fatal(err)
			}
			after, err := time.Parse(time.RFC3339, tt.first)
			if err != nil {
				t.Fatal(err)
			}
			wantHijri(t, cal, before, timex.Jakarta, tt.before)
			wantHijri(t, cal, after, timex.Jakarta, tt.after)
		})
	}
}

func TestUmmAlQura_SameInstantDiffersByLocation(t *testing.T) {
	cal := newUmmAlQura(t)
	instant := time.Date(2026, 6, 15, 17, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		loc  *time.Location
		want timex.HijriDate
	}{
		{"UTC is still the old year", time.UTC, timex.HijriDate{Year: 1447, Month: 12, Day: 29}},
		{"WIB is the new year", time.FixedZone("WIB", 7*3600), timex.HijriDate{Year: 1448, Month: 1, Day: 1}},
		{"one second short of midnight", time.FixedZone("x", 7*3600-1), timex.HijriDate{Year: 1447, Month: 12, Day: 29}},
		{"WIT is the new year", timex.Jayapura, timex.HijriDate{Year: 1448, Month: 1, Day: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantHijri(t, cal, instant, tt.loc, tt.want)
		})
	}
}

func TestUmmAlQura_DaylightSavingLocation(t *testing.T) {
	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	cal := newUmmAlQura(t)
	// 1 Muharram 1448 H is 16 June 2026. London is on BST (+01:00), so the day
	// begins at 23:00 UTC on the 15th, not at 00:00 UTC.
	wantHijri(t, cal, time.Date(2026, 6, 15, 22, 59, 59, 0, time.UTC), london, timex.HijriDate{Year: 1447, Month: 12, Day: 29})
	wantHijri(t, cal, time.Date(2026, 6, 15, 23, 0, 0, 0, time.UTC), london, timex.HijriDate{Year: 1448, Month: 1, Day: 1})
}

// TestUmmAlQura_VanGentPrincipalDays checks the table against the five dates per
// year in R.H. van Gent, "The Principal Islamic Days of Observance according to
// the Umm al-Qura Calendar", which lists 1420-1450 H:
// https://webspace.science.uu.nl/~gent0113/islam/ummalqura_principal.htm
func TestUmmAlQura_VanGentPrincipalDays(t *testing.T) {
	cal := newUmmAlQura(t)
	principal := [5]struct{ month, day int }{{1, 1}, {3, 12}, {9, 1}, {10, 1}, {12, 10}}
	years := []struct {
		year  int
		dates [5]string
	}{
		{1420, [5]string{"1999-04-17", "1999-06-26", "1999-12-09", "2000-01-08", "2000-03-16"}},
		{1421, [5]string{"2000-04-06", "2000-06-14", "2000-11-27", "2000-12-27", "2001-03-05"}},
		{1422, [5]string{"2001-03-26", "2001-06-04", "2001-11-16", "2001-12-16", "2002-02-22"}},
		{1423, [5]string{"2002-03-15", "2002-05-24", "2002-11-06", "2002-12-05", "2003-02-11"}},
		{1424, [5]string{"2003-03-04", "2003-05-13", "2003-10-26", "2003-11-25", "2004-02-01"}},
		{1425, [5]string{"2004-02-21", "2004-05-01", "2004-10-15", "2004-11-14", "2005-01-21"}},
		{1426, [5]string{"2005-02-10", "2005-04-21", "2005-10-04", "2005-11-03", "2006-01-10"}},
		{1427, [5]string{"2006-01-31", "2006-04-10", "2006-09-24", "2006-10-23", "2006-12-31"}},
		{1428, [5]string{"2007-01-20", "2007-03-31", "2007-09-13", "2007-10-13", "2007-12-20"}},
		{1429, [5]string{"2008-01-10", "2008-03-20", "2008-09-01", "2008-10-01", "2008-12-08"}},
		{1430, [5]string{"2008-12-29", "2009-03-09", "2009-08-22", "2009-09-20", "2009-11-27"}},
		{1431, [5]string{"2009-12-18", "2010-02-26", "2010-08-11", "2010-09-10", "2010-11-16"}},
		{1432, [5]string{"2010-12-07", "2011-02-15", "2011-08-01", "2011-08-30", "2011-11-06"}},
		{1433, [5]string{"2011-11-26", "2012-02-04", "2012-07-20", "2012-08-19", "2012-10-26"}},
		{1434, [5]string{"2012-11-15", "2013-01-24", "2013-07-09", "2013-08-08", "2013-10-15"}},
		{1435, [5]string{"2013-11-04", "2014-01-13", "2014-06-28", "2014-07-28", "2014-10-04"}},
		{1436, [5]string{"2014-10-25", "2015-01-03", "2015-06-18", "2015-07-17", "2015-09-23"}},
		{1437, [5]string{"2015-10-14", "2015-12-23", "2016-06-06", "2016-07-06", "2016-09-11"}},
		{1438, [5]string{"2016-10-02", "2016-12-11", "2017-05-27", "2017-06-25", "2017-09-01"}},
		{1439, [5]string{"2017-09-21", "2017-11-30", "2018-05-16", "2018-06-15", "2018-08-21"}},
		{1440, [5]string{"2018-09-11", "2018-11-20", "2019-05-06", "2019-06-04", "2019-08-11"}},
		{1441, [5]string{"2019-08-31", "2019-11-09", "2020-04-24", "2020-05-24", "2020-07-31"}},
		{1442, [5]string{"2020-08-20", "2020-10-29", "2021-04-13", "2021-05-13", "2021-07-20"}},
		{1443, [5]string{"2021-08-09", "2021-10-18", "2022-04-02", "2022-05-02", "2022-07-09"}},
		{1444, [5]string{"2022-07-30", "2022-10-08", "2023-03-23", "2023-04-21", "2023-06-28"}},
		{1445, [5]string{"2023-07-19", "2023-09-27", "2024-03-11", "2024-04-10", "2024-06-16"}},
		{1446, [5]string{"2024-07-07", "2024-09-15", "2025-03-01", "2025-03-30", "2025-06-06"}},
		{1447, [5]string{"2025-06-26", "2025-09-04", "2026-02-18", "2026-03-20", "2026-05-27"}},
		{1448, [5]string{"2026-06-16", "2026-08-25", "2027-02-08", "2027-03-09", "2027-05-16"}},
		{1449, [5]string{"2027-06-06", "2027-08-14", "2028-01-28", "2028-02-26", "2028-05-05"}},
		{1450, [5]string{"2028-05-25", "2028-08-03", "2029-01-16", "2029-02-14", "2029-04-24"}},
	}
	if got := len(years); got != 31 {
		t.Fatalf("van Gent table has %d years, want 31", got)
	}
	for _, y := range years {
		for i, date := range y.dates {
			want := timex.HijriDate{Year: y.year, Month: principal[i].month, Day: principal[i].day}
			wantHijri(t, cal, civilDate(t, date, time.UTC), time.UTC, want)
		}
	}
}

// TestUmmAlQura_TableMatchesICUSource pins the embedded table. The checksum is
// the SHA-256 of one length byte per month, 1356-1500 H, decoded from
// UMALQURA_MONTHLENGTH in ICU release-76-1 outside this module.
func TestUmmAlQura_TableMatchesICUSource(t *testing.T) {
	cal := newUmmAlQura(t)
	lengths := make([]byte, 0, 145*12)
	for year := 1356; year <= 1500; year++ {
		for month := 1; month <= 12; month++ {
			n := cal.MonthLength(year, month)
			if n != 29 && n != 30 {
				t.Fatalf("MonthLength(%d, %d) = %d, want 29 or 30", year, month, n)
			}
			lengths = append(lengths, byte(n)) //nolint:gosec // n is 29 or 30, checked above
		}
	}
	sum := sha256.Sum256(lengths)
	const want = "b61d78db78a9673338c5f810ff5c11c0ca1d9a9026c9d91cc365c0bb3f3911e2"
	if got := hex.EncodeToString(sum[:]); got != want {
		t.Errorf("table checksum = %s, want %s", got, want)
	}
}

func TestUmmAlQura_TableBounds(t *testing.T) {
	cal := newUmmAlQura(t)
	valid := []struct {
		name string
		at   time.Time
		want timex.HijriDate
	}{
		{"first day", time.Date(1937, 3, 14, 0, 0, 0, 0, time.UTC), timex.HijriDate{Year: 1356, Month: 1, Day: 1}},
		{"end of first day", time.Date(1937, 3, 14, 23, 59, 59, 999999999, time.UTC), timex.HijriDate{Year: 1356, Month: 1, Day: 1}},
		{"last day", time.Date(2077, 11, 16, 0, 0, 0, 0, time.UTC), timex.HijriDate{Year: 1500, Month: 12, Day: 30}},
		{"end of last day", time.Date(2077, 11, 16, 23, 59, 59, 999999999, time.UTC), timex.HijriDate{Year: 1500, Month: 12, Day: 30}},
	}
	for _, tt := range valid {
		t.Run(tt.name, func(t *testing.T) {
			wantHijri(t, cal, tt.at, time.UTC, tt.want)
		})
	}

	outOfRange := []struct {
		name string
		at   time.Time
		loc  *time.Location
	}{
		{"day before the table", time.Date(1937, 3, 13, 23, 59, 59, 999999999, time.UTC), time.UTC},
		{"day after the table", time.Date(2077, 11, 17, 0, 0, 0, 0, time.UTC), time.UTC},
		{"zero time", time.Time{}, time.UTC},
		{"civil year before the guard", time.Date(1936, 12, 31, 0, 0, 0, 0, time.UTC), time.UTC},
		{"civil year after the guard", time.Date(2078, 1, 1, 0, 0, 0, 0, time.UTC), time.UTC},
		{"far past", time.Unix(-1<<62, 0), time.UTC},
		{"far future", time.Unix(1<<62, 0), time.UTC},
		{"in range in UTC, after the table locally", time.Date(2077, 11, 16, 20, 0, 0, 0, time.UTC), time.FixedZone("WIB", 7*3600)},
		{"in range in UTC, before the table locally", time.Date(1937, 3, 14, 3, 0, 0, 0, time.UTC), time.FixedZone("x", -5*3600)},
	}
	for _, tt := range outOfRange {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cal.FromCivilDate(tt.at, tt.loc)
			if !errors.Is(err, timex.ErrHijriOutOfRange) {
				t.Fatalf("err = %v, want ErrHijriOutOfRange", err)
			}
			if got != (timex.HijriDate{}) {
				t.Errorf("date = %+v, want zero value", got)
			}
		})
	}
}

func TestUmmAlQura_NilLocation(t *testing.T) {
	cal := newUmmAlQura(t)
	_, err := cal.FromCivilDate(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), nil)
	if err == nil {
		t.Fatal("want an error for a nil location")
	}
	if errors.Is(err, timex.ErrHijriOutOfRange) {
		t.Errorf("err = %v, must not be ErrHijriOutOfRange", err)
	}
}

func TestUmmAlQura_MonthLength(t *testing.T) {
	cal := newUmmAlQura(t)
	tests := []struct {
		name        string
		year, month int
		want        int
	}{
		{"29-day month", 1447, 12, 29},
		{"30-day month", 1448, 5, 30},
		{"29-day Dzulhijjah 1446", 1446, 12, 29},
		{"30-day Ramadhan 1447", 1447, 9, 30},
		{"first table month", 1356, 1, 29},
		{"last table month", 1500, 12, 30},
		{"year before the table", 1355, 12, 0},
		{"year after the table", 1501, 1, 0},
		{"month zero", 1448, 0, 0},
		{"month thirteen", 1448, 13, 0},
		{"negative month", 1448, -1, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cal.MonthLength(tt.year, tt.month); got != tt.want {
				t.Errorf("MonthLength(%d, %d) = %d, want %d", tt.year, tt.month, got, tt.want)
			}
		})
	}
}

// TestUmmAlQura_MonthLengthsDriveConversion ties MonthLength to FromCivilDate:
// each month has exactly MonthLength days, and every civil day is covered once.
func TestUmmAlQura_MonthLengthsDriveConversion(t *testing.T) {
	offsets := timex.WithHijriMonthOffsets(
		timex.HijriMonthOffset{Year: 1447, Month: 9, Days: 1},
		timex.HijriMonthOffset{Year: 1447, Month: 10, Days: 1},
		timex.HijriMonthOffset{Year: 1447, Month: 11, Days: 1},
	)
	tests := []struct {
		name  string
		cal   *timex.UmmAlQura
		first time.Time
	}{
		{"table", newUmmAlQura(t), time.Date(1937, 3, 14, 0, 0, 0, 0, time.UTC)},
		{"with offsets", newUmmAlQura(t, offsets), time.Date(1937, 3, 14, 0, 0, 0, 0, time.UTC)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			day := tt.first
			for year := 1356; year <= 1500; year++ {
				for month := 1; month <= 12; month++ {
					length := tt.cal.MonthLength(year, month)
					for d := 1; d <= length; d++ {
						wantHijri(t, tt.cal, day, time.UTC, timex.HijriDate{Year: year, Month: month, Day: d})
						day = day.AddDate(0, 0, 1)
					}
				}
			}
			if want := time.Date(2077, 11, 17, 0, 0, 0, 0, time.UTC); !day.Equal(want) {
				t.Errorf("walk ended at %s, want %s", day.Format(time.DateOnly), want.Format(time.DateOnly))
			}
		})
	}
}

func TestWithHijriMonthOffsets_KemenagDates(t *testing.T) {
	type check struct {
		date string
		want timex.HijriDate
	}
	tests := []struct {
		name    string
		offsets []timex.HijriMonthOffset
		checks  []check
	}{
		{
			name:    "Syawal 1446 starts a day later",
			offsets: []timex.HijriMonthOffset{{Year: 1446, Month: 10, Days: 1}},
			checks: []check{
				{"2025-03-30", timex.HijriDate{Year: 1446, Month: 9, Day: 30}},
				{"2025-03-31", timex.HijriDate{Year: 1446, Month: 10, Day: 1}},
				{"2025-03-01", timex.HijriDate{Year: 1446, Month: 9, Day: 1}},
			},
		},
		{
			name: "Ramadhan through Dzulqa'dah 1447 start a day later",
			offsets: []timex.HijriMonthOffset{
				{Year: 1447, Month: 9, Days: 1},
				{Year: 1447, Month: 10, Days: 1},
				{Year: 1447, Month: 11, Days: 1},
			},
			checks: []check{
				{"2026-02-18", timex.HijriDate{Year: 1447, Month: 8, Day: 30}},
				{"2026-02-19", timex.HijriDate{Year: 1447, Month: 9, Day: 1}},
				{"2026-03-19", timex.HijriDate{Year: 1447, Month: 9, Day: 29}},
				{"2026-03-20", timex.HijriDate{Year: 1447, Month: 9, Day: 30}},
				{"2026-03-21", timex.HijriDate{Year: 1447, Month: 10, Day: 1}},
				{"2026-05-18", timex.HijriDate{Year: 1447, Month: 12, Day: 1}},
				{"2026-05-27", timex.HijriDate{Year: 1447, Month: 12, Day: 10}},
			},
		},
		{
			name:    "Rabiul Akhir 1448 starts a day later",
			offsets: []timex.HijriMonthOffset{{Year: 1448, Month: 4, Days: 1}},
			checks: []check{
				{"2026-09-12", timex.HijriDate{Year: 1448, Month: 3, Day: 30}},
				{"2026-10-11", timex.HijriDate{Year: 1448, Month: 4, Day: 29}},
				{"2026-10-12", timex.HijriDate{Year: 1448, Month: 5, Day: 1}},
				{"2026-10-28", timex.HijriDate{Year: 1448, Month: 5, Day: 17}},
			},
		},
		{
			name:    "a month starts a day earlier",
			offsets: []timex.HijriMonthOffset{{Year: 1448, Month: 6, Days: -1}},
			checks: []check{
				{"2026-11-09", timex.HijriDate{Year: 1448, Month: 5, Day: 29}},
				{"2026-11-10", timex.HijriDate{Year: 1448, Month: 6, Day: 1}},
				{"2026-11-11", timex.HijriDate{Year: 1448, Month: 6, Day: 2}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cal := newUmmAlQura(t, timex.WithHijriMonthOffsets(tt.offsets...))
			for _, c := range tt.checks {
				wantHijri(t, cal, civilDate(t, c.date, timex.Jakarta), timex.Jakarta, c.want)
			}
		})
	}
}

func TestWithHijriMonthOffsets_MonthLengths(t *testing.T) {
	cal := newUmmAlQura(t, timex.WithHijriMonthOffsets(timex.HijriMonthOffset{Year: 1448, Month: 6, Days: -1}))
	if got := cal.MonthLength(1448, 5); got != 29 {
		t.Errorf("preceding month = %d, want 29", got)
	}
	if got := cal.MonthLength(1448, 6); got != 30 {
		t.Errorf("shifted month = %d, want 30", got)
	}

	base := newUmmAlQura(t)
	if got, want := cal.MonthLength(1448, 7), base.MonthLength(1448, 7); got != want {
		t.Errorf("following month = %d, want the unchanged %d", got, want)
	}
}

func TestWithHijriMonthOffsets_Repeatable(t *testing.T) {
	cal := newUmmAlQura(t,
		timex.WithHijriMonthOffsets(timex.HijriMonthOffset{Year: 1446, Month: 10, Days: 1}),
		timex.WithHijriMonthOffsets(timex.HijriMonthOffset{Year: 1448, Month: 4, Days: 1}),
	)
	wantHijri(t, cal, civilDate(t, "2025-03-30", time.UTC), time.UTC, timex.HijriDate{Year: 1446, Month: 9, Day: 30})
	wantHijri(t, cal, civilDate(t, "2026-10-11", time.UTC), time.UTC, timex.HijriDate{Year: 1448, Month: 4, Day: 29})
}

func TestWithHijriMonthOffsets_DoesNotKeepCallerSlice(t *testing.T) {
	offsets := []timex.HijriMonthOffset{{Year: 1446, Month: 10, Days: 1}}
	opt := timex.WithHijriMonthOffsets(offsets...)
	offsets[0] = timex.HijriMonthOffset{Year: 1, Month: 1, Days: 9}
	cal := newUmmAlQura(t, opt)
	wantHijri(t, cal, civilDate(t, "2025-03-30", time.UTC), time.UTC, timex.HijriDate{Year: 1446, Month: 9, Day: 30})
}

func TestWithHijriMonthOffsets_TableEdges(t *testing.T) {
	t.Run("first month earlier extends the table by a day", func(t *testing.T) {
		cal := newUmmAlQura(t, timex.WithHijriMonthOffsets(timex.HijriMonthOffset{Year: 1356, Month: 1, Days: -1}))
		wantHijri(t, cal, time.Date(1937, 3, 13, 0, 0, 0, 0, time.UTC), time.UTC, timex.HijriDate{Year: 1356, Month: 1, Day: 1})
		if _, err := cal.FromCivilDate(time.Date(1937, 3, 12, 0, 0, 0, 0, time.UTC), time.UTC); !errors.Is(err, timex.ErrHijriOutOfRange) {
			t.Errorf("err = %v, want ErrHijriOutOfRange", err)
		}
	})
	t.Run("last month later ends at the same day", func(t *testing.T) {
		cal := newUmmAlQura(t, timex.WithHijriMonthOffsets(
			timex.HijriMonthOffset{Year: 1500, Month: 11, Days: 1},
			timex.HijriMonthOffset{Year: 1500, Month: 12, Days: 1},
		))
		if got := cal.MonthLength(1500, 12); got != 29 {
			t.Errorf("MonthLength = %d, want 29", got)
		}
		wantHijri(t, cal, time.Date(2077, 11, 16, 0, 0, 0, 0, time.UTC), time.UTC, timex.HijriDate{Year: 1500, Month: 12, Day: 29})
		if _, err := cal.FromCivilDate(time.Date(2077, 11, 17, 0, 0, 0, 0, time.UTC), time.UTC); !errors.Is(err, timex.ErrHijriOutOfRange) {
			t.Errorf("err = %v, want ErrHijriOutOfRange", err)
		}
	})
}

func TestNewUmmAlQura_RejectsOffsets(t *testing.T) {
	tests := []struct {
		name       string
		offsets    []timex.HijriMonthOffset
		errContain string
		outOfRange bool
	}{
		{
			name: "shifted start leaves a 28-day month",
			offsets: []timex.HijriMonthOffset{
				{Year: 1447, Month: 9, Days: 1},
				{Year: 1447, Month: 10, Days: 1},
			},
			errContain: "month 10 of 1447 H 28 days long",
		},
		{
			name:       "shifted start stretches the previous month to 31 days",
			offsets:    []timex.HijriMonthOffset{{Year: 1448, Month: 6, Days: 1}},
			errContain: "month 5 of 1448 H 31 days long",
		},
		{
			name:       "shifted start shortens the previous month to 28 days",
			offsets:    []timex.HijriMonthOffset{{Year: 1447, Month: 11, Days: -1}},
			errContain: "month 10 of 1447 H 28 days long",
		},
		{
			name:       "last offset month becomes 31 days",
			offsets:    []timex.HijriMonthOffset{{Year: 1500, Month: 12, Days: -1}},
			errContain: "month 12 of 1500 H 31 days long",
		},
		{
			name:       "first table month becomes 28 days",
			offsets:    []timex.HijriMonthOffset{{Year: 1356, Month: 1, Days: 1}},
			errContain: "month 1 of 1356 H 28 days long",
		},
		{
			name:       "zero days",
			offsets:    []timex.HijriMonthOffset{{Year: 1448, Month: 7}},
			errContain: "days 0 is not -1 or 1",
		},
		{
			name:       "two days",
			offsets:    []timex.HijriMonthOffset{{Year: 1448, Month: 7, Days: 2}},
			errContain: "days 2 is not -1 or 1",
		},
		{
			name:       "month zero",
			offsets:    []timex.HijriMonthOffset{{Year: 1448, Month: 0, Days: 1}},
			errContain: "month 0 is not 1 to 12",
		},
		{
			name:       "month thirteen",
			offsets:    []timex.HijriMonthOffset{{Year: 1448, Month: 13, Days: 1}},
			errContain: "month 13 is not 1 to 12",
		},
		{
			name: "same month twice",
			offsets: []timex.HijriMonthOffset{
				{Year: 1446, Month: 10, Days: 1},
				{Year: 1446, Month: 10, Days: 1},
			},
			errContain: "given twice",
		},
		{
			name:       "year before the table",
			offsets:    []timex.HijriMonthOffset{{Year: 1355, Month: 12, Days: 1}},
			errContain: "1355",
			outOfRange: true,
		},
		{
			name:       "year after the table",
			offsets:    []timex.HijriMonthOffset{{Year: 1501, Month: 1, Days: 1}},
			errContain: "1501",
			outOfRange: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cal, err := timex.NewUmmAlQura(timex.WithHijriMonthOffsets(tt.offsets...))
			if err == nil {
				t.Fatal("want an error")
			}
			if cal != nil {
				t.Error("calendar must be nil on error")
			}
			if !strings.Contains(err.Error(), tt.errContain) {
				t.Errorf("err = %q, want it to contain %q", err, tt.errContain)
			}
			if got := errors.Is(err, timex.ErrHijriOutOfRange); got != tt.outOfRange {
				t.Errorf("errors.Is(ErrHijriOutOfRange) = %v, want %v", got, tt.outOfRange)
			}
		})
	}
}

func TestNewUmmAlQura_RejectsDuplicateAcrossOptions(t *testing.T) {
	o := timex.HijriMonthOffset{Year: 1446, Month: 10, Days: 1}
	if _, err := timex.NewUmmAlQura(timex.WithHijriMonthOffsets(o), timex.WithHijriMonthOffsets(o)); err == nil {
		t.Fatal("want an error for a month given in two options")
	}
}

func TestUmmAlQura_ConcurrentUse(t *testing.T) {
	cal := newUmmAlQura(t, timex.WithHijriMonthOffsets(timex.HijriMonthOffset{Year: 1446, Month: 10, Days: 1}))
	want := timex.HijriDate{Year: 1446, Month: 9, Day: 30}
	at := time.Date(2025, 3, 30, 12, 0, 0, 0, timex.Jakarta)

	var wg sync.WaitGroup
	results := make([]timex.HijriDate, 32)
	for i := range results {
		wg.Go(func() {
			got, err := cal.FromCivilDate(at, timex.Jakarta)
			if err != nil {
				t.Error(err)
			}
			results[i] = got
		})
	}
	wg.Wait()
	if !slices.Equal(results, slices.Repeat([]timex.HijriDate{want}, len(results))) {
		t.Errorf("results = %v, want all %+v", results, want)
	}
}
