package timex_test

import (
	"errors"
	"testing"
	"time"

	"github.com/santekno/sdk/timex"
)

func TestUmmAlQura_ToCivilDate_ReferenceDates(t *testing.T) {
	cal := newUmmAlQura(t)
	for _, tc := range hijriReferenceDates {
		t.Run(tc.date, func(t *testing.T) {
			got, err := cal.ToCivilDate(tc.want, timex.Jakarta)
			if err != nil {
				t.Fatalf("ToCivilDate(%+v): %v", tc.want, err)
			}
			want := civilDate(t, tc.date, timex.Jakarta)
			if !got.Equal(want) || got.Location() != timex.Jakarta {
				t.Fatalf("ToCivilDate(%+v) = %v, want %v", tc.want, got, want)
			}
		})
	}
}

// Every day of the table, with and without offsets, round-trips through
// FromCivilDate.
func TestUmmAlQura_ToCivilDate_InverseOfFromCivilDate(t *testing.T) {
	tests := []struct {
		name string
		opts []timex.UmmAlQuraOption
	}{
		{name: "table"},
		{name: "offsets", opts: []timex.UmmAlQuraOption{timex.WithHijriMonthOffsets(
			timex.HijriMonthOffset{Year: 1356, Month: 1, Days: -1},
			timex.HijriMonthOffset{Year: 1446, Month: 10, Days: 1},
			timex.HijriMonthOffset{Year: 1448, Month: 7, Days: 1},
		)}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cal := newUmmAlQura(t, tc.opts...)
			var prev time.Time
			for year := 1356; year <= 1500; year++ {
				for month := 1; month <= 12; month++ {
					n := cal.MonthLength(year, month)
					for day := 1; day <= n; day++ {
						d := timex.HijriDate{Year: year, Month: month, Day: day}
						civil, err := cal.ToCivilDate(d, time.UTC)
						if err != nil {
							t.Fatalf("ToCivilDate(%+v): %v", d, err)
						}
						if !prev.IsZero() && !civil.Equal(prev.AddDate(0, 0, 1)) {
							t.Fatalf("ToCivilDate(%+v) = %v, want the day after %v", d, civil, prev)
						}
						prev = civil
						back, err := cal.FromCivilDate(civil, time.UTC)
						if err != nil || back != d {
							t.Fatalf("FromCivilDate(ToCivilDate(%+v)) = %+v, %v", d, back, err)
						}
					}
				}
			}
		})
	}
}

func TestUmmAlQura_ToCivilDate_Offsets(t *testing.T) {
	tests := []struct {
		name   string
		offset timex.HijriMonthOffset
		date   timex.HijriDate
		want   string
	}{
		{"later start", timex.HijriMonthOffset{Year: 1448, Month: 7, Days: 1}, timex.HijriDate{Year: 1448, Month: 7, Day: 1}, "2026-12-11"},
		{"earlier start", timex.HijriMonthOffset{Year: 1448, Month: 6, Days: -1}, timex.HijriDate{Year: 1448, Month: 6, Day: 1}, "2026-11-10"},
		{"previous month absorbs", timex.HijriMonthOffset{Year: 1448, Month: 7, Days: 1}, timex.HijriDate{Year: 1448, Month: 6, Day: 30}, "2026-12-10"},
		{"next month unchanged", timex.HijriMonthOffset{Year: 1448, Month: 7, Days: 1}, timex.HijriDate{Year: 1448, Month: 8, Day: 1}, "2027-01-09"},
		{"first table month", timex.HijriMonthOffset{Year: 1356, Month: 1, Days: -1}, timex.HijriDate{Year: 1356, Month: 1, Day: 1}, "1937-03-13"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cal := newUmmAlQura(t, timex.WithHijriMonthOffsets(tc.offset))
			got, err := cal.ToCivilDate(tc.date, timex.Jakarta)
			if err != nil {
				t.Fatalf("ToCivilDate: %v", err)
			}
			if s := got.Format(time.DateOnly); s != tc.want {
				t.Fatalf("ToCivilDate(%+v) = %s, want %s", tc.date, s, tc.want)
			}
		})
	}
}

func TestUmmAlQura_ToCivilDate_TableBounds(t *testing.T) {
	cal := newUmmAlQura(t)
	tests := []struct {
		date timex.HijriDate
		want string
	}{
		{timex.HijriDate{Year: 1356, Month: 1, Day: 1}, "1937-03-14"},
		{timex.HijriDate{Year: 1500, Month: 12, Day: cal.MonthLength(1500, 12)}, "2077-11-16"},
	}
	for _, tc := range tests {
		got, err := cal.ToCivilDate(tc.date, time.UTC)
		if err != nil {
			t.Fatalf("ToCivilDate(%+v): %v", tc.date, err)
		}
		if s := got.Format(time.DateOnly); s != tc.want {
			t.Fatalf("ToCivilDate(%+v) = %s, want %s", tc.date, s, tc.want)
		}
	}
}

func TestUmmAlQura_ToCivilDate_Rejects(t *testing.T) {
	cal := newUmmAlQura(t)
	tests := []struct {
		name string
		date timex.HijriDate
		loc  *time.Location
		want error
	}{
		{"nil location", timex.HijriDate{Year: 1448, Month: 1, Day: 1}, nil, nil},
		{"year before table", timex.HijriDate{Year: 1355, Month: 12, Day: 1}, time.UTC, timex.ErrHijriOutOfRange},
		{"year after table", timex.HijriDate{Year: 1501, Month: 1, Day: 1}, time.UTC, timex.ErrHijriOutOfRange},
		{"month zero", timex.HijriDate{Year: 1448, Month: 0, Day: 1}, time.UTC, timex.ErrInvalidHijriDate},
		{"month thirteen", timex.HijriDate{Year: 1448, Month: 13, Day: 1}, time.UTC, timex.ErrInvalidHijriDate},
		{"day zero", timex.HijriDate{Year: 1448, Month: 1, Day: 0}, time.UTC, timex.ErrInvalidHijriDate},
		{"day 30 of a 29-day month", timex.HijriDate{Year: 1448, Month: 6, Day: 30}, time.UTC, timex.ErrInvalidHijriDate},
		{"day 31", timex.HijriDate{Year: 1448, Month: 5, Day: 31}, time.UTC, timex.ErrInvalidHijriDate},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := cal.ToCivilDate(tc.date, tc.loc)
			if err == nil {
				t.Fatalf("ToCivilDate(%+v) = %v, want error", tc.date, got)
			}
			if !got.IsZero() {
				t.Fatalf("ToCivilDate(%+v) returned %v with an error", tc.date, got)
			}
			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("ToCivilDate(%+v) error = %v, want %v", tc.date, err, tc.want)
			}
			if tc.want == nil && (errors.Is(err, timex.ErrHijriOutOfRange) || errors.Is(err, timex.ErrInvalidHijriDate)) {
				t.Fatalf("nil location error %v should not match a date sentinel", err)
			}
		})
	}
}

func TestNewUmmAlQura_MonthLengthError(t *testing.T) {
	tests := []struct {
		name    string
		offsets []timex.HijriMonthOffset
		want    timex.HijriMonthLengthError
	}{
		{"previous month 31", []timex.HijriMonthOffset{{Year: 1448, Month: 6, Days: 1}}, timex.HijriMonthLengthError{Year: 1448, Month: 5, Days: 31}},
		{"previous month 28", []timex.HijriMonthOffset{{Year: 1448, Month: 7, Days: -1}}, timex.HijriMonthLengthError{Year: 1448, Month: 6, Days: 28}},
		{"two offsets make 31", []timex.HijriMonthOffset{{Year: 1448, Month: 9, Days: -1}, {Year: 1448, Month: 10, Days: 1}}, timex.HijriMonthLengthError{Year: 1448, Month: 9, Days: 31}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := timex.NewUmmAlQura(timex.WithHijriMonthOffsets(tc.offsets...))
			var got *timex.HijriMonthLengthError
			if !errors.As(err, &got) {
				t.Fatalf("NewUmmAlQura error = %v, want *HijriMonthLengthError", err)
			}
			if *got != tc.want {
				t.Fatalf("HijriMonthLengthError = %+v, want %+v", *got, tc.want)
			}
			if got.Error() != err.Error() || got.Error() == "" {
				t.Fatalf("Error() = %q", got.Error())
			}
		})
	}
}

func TestNewUmmAlQura_OtherErrorsAreNotMonthLength(t *testing.T) {
	tests := [][]timex.HijriMonthOffset{
		{{Year: 1448, Month: 13, Days: 1}},
		{{Year: 1448, Month: 7, Days: 2}},
		{{Year: 1501, Month: 1, Days: 1}},
		{{Year: 1448, Month: 7, Days: 1}, {Year: 1448, Month: 7, Days: 1}},
	}
	for _, offsets := range tests {
		_, err := timex.NewUmmAlQura(timex.WithHijriMonthOffsets(offsets...))
		var got *timex.HijriMonthLengthError
		if err == nil || errors.As(err, &got) {
			t.Fatalf("offsets %+v: error = %v, want a non-length error", offsets, err)
		}
	}
}
