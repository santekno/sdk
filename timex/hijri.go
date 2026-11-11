package timex

import (
	"errors"
	"fmt"
	"slices"
	"time"
)

// Gregorian years that safely enclose the Umm al-Qura table, offsets included.
// FromCivilDate rejects anything outside them before doing day arithmetic, so
// extreme time.Time values cannot overflow into a valid-looking date.
const (
	ummAlQuraMinCivilYear = 1937
	ummAlQuraMaxCivilYear = 2077

	secondsPerDay = 86400
)

// ErrHijriOutOfRange is returned when a date or a month offset falls outside
// the embedded Umm al-Qura table, which covers 1356-1500 H (14 March 1937 to
// 16 November 2077 without offsets). Dates outside the table are never
// estimated.
var ErrHijriOutOfRange = errors.New("timex: outside the Umm al-Qura table (1356-1500 H)")

// ErrInvalidHijriDate is returned by ToCivilDate when a month is not 1 to 12 or
// a day is not between 1 and the length of its month.
var ErrInvalidHijriDate = errors.New("timex: invalid Hijri date")

// HijriMonthLengthError is returned by NewUmmAlQura when month offsets would
// leave a month with a length other than 29 or 30 days. Its fields let callers
// explain the rejection without parsing the message.
type HijriMonthLengthError struct {
	// Year is the Hijri year of the month that changed length.
	Year int
	// Month is the Hijri month, 1 to 12.
	Month int
	// Days is the length the month would have, offsets applied.
	Days int
}

// Error implements error.
func (e *HijriMonthLengthError) Error() string {
	return fmt.Sprintf("timex: Hijri month offsets make month %d of %d H %d days long, want 29 or 30", e.Month, e.Year, e.Days)
}

// HijriDate is a date in the Hijri calendar.
type HijriDate struct {
	// Year is the Hijri year, e.g. 1448.
	Year int
	// Month is the Hijri month: 1 is Muharram and 12 is Dzulhijjah.
	Month int
	// Day is the day of the month, 1 to 30.
	Day int
}

// HijriMonthOffset moves the first day of one Hijri month relative to the
// Umm al-Qura table. It exists for authorities that fix months by sighting and
// can differ from the table by a day.
//
// An offset affects that month's start only. The preceding month absorbs the
// change, and later months keep their table dates unless they carry an offset
// of their own.
type HijriMonthOffset struct {
	// Year is the Hijri year, 1356 to 1500.
	Year int
	// Month is the Hijri month, 1 to 12.
	Month int
	// Days is -1 to start the month a day earlier or 1 to start it a day later.
	Days int
}

// UmmAlQuraOption configures NewUmmAlQura.
type UmmAlQuraOption func(*ummAlQuraConfig)

type ummAlQuraConfig struct {
	offsets []HijriMonthOffset
}

// WithHijriMonthOffsets applies month offsets to the calendar. The option can
// be repeated; offsets accumulate, and each month may appear only once overall.
func WithHijriMonthOffsets(offsets ...HijriMonthOffset) UmmAlQuraOption {
	offsets = slices.Clone(offsets)
	return func(c *ummAlQuraConfig) { c.offsets = append(c.offsets, offsets...) }
}

// UmmAlQura converts Gregorian civil dates to the Umm al-Qura calendar. It is
// immutable after NewUmmAlQura and safe for concurrent use.
type UmmAlQura struct {
	// starts holds the first day of each month in days since 1970-01-01, offsets
	// applied, followed by the day after the last month. It is strictly
	// increasing.
	starts [ummAlQuraMonths + 1]int
}

// NewUmmAlQura builds a calendar from the embedded table and any offsets.
//
// It returns an error if an offset names an invalid month or a Days value other
// than -1 or 1, if two offsets name the same month, or if any month would no
// longer be 29 or 30 days long (a *HijriMonthLengthError naming the first such
// month). Offsets for a year outside 1356-1500 H wrap
// ErrHijriOutOfRange. The first and last table months can only be shifted within
// these rules, and the end of Dzulhijjah 1500 H cannot move.
func NewUmmAlQura(opts ...UmmAlQuraOption) (*UmmAlQura, error) {
	var cfg ummAlQuraConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	shifts, err := hijriMonthShifts(cfg.offsets)
	if err != nil {
		return nil, err
	}

	c := &UmmAlQura{}
	day := ummAlQuraFirstDay
	for i := range ummAlQuraMonths {
		c.starts[i] = day + shifts[i]
		day += ummAlQuraBaseLength(i)
	}
	c.starts[ummAlQuraMonths] = day

	for i := range ummAlQuraMonths {
		if n := c.starts[i+1] - c.starts[i]; n != 29 && n != 30 {
			year, month := hijriMonthAt(i)
			return nil, &HijriMonthLengthError{Year: year, Month: month, Days: n}
		}
	}
	return c, nil
}

// FromCivilDate returns the Hijri date of the calendar day t falls on in loc.
//
// Only the year, month, and day of t in loc are used, and t's own location is
// ignored. The Hijri day therefore changes at 00:00 in loc, not at sunset. A nil
// loc is an error rather than a default. Dates outside the table return
// ErrHijriOutOfRange.
func (c *UmmAlQura) FromCivilDate(t time.Time, loc *time.Location) (HijriDate, error) {
	if loc == nil {
		return HijriDate{}, errors.New("timex: FromCivilDate requires a location")
	}

	year, month, day := t.In(loc).Date()
	if year < ummAlQuraMinCivilYear || year > ummAlQuraMaxCivilYear {
		return HijriDate{}, hijriDateOutOfRange(year, month, day)
	}

	civil := int(time.Date(year, month, day, 0, 0, 0, 0, time.UTC).Unix() / secondsPerDay)
	if civil < c.starts[0] || civil >= c.starts[ummAlQuraMonths] {
		return HijriDate{}, hijriDateOutOfRange(year, month, day)
	}

	i, found := slices.BinarySearch(c.starts[:], civil)
	if !found {
		i--
	}
	hijriYear, hijriMonth := hijriMonthAt(i)
	return HijriDate{Year: hijriYear, Month: hijriMonth, Day: civil - c.starts[i] + 1}, nil
}

// ToCivilDate returns the Gregorian civil date of a Hijri date, offsets
// applied, as 00:00 in loc. It is the inverse of FromCivilDate: for every date
// in the table, FromCivilDate(ToCivilDate(d, loc), loc) returns d.
//
// A nil loc is an error rather than a default. A year outside 1356-1500 H wraps
// ErrHijriOutOfRange; a month other than 1 to 12, or a day outside 1 to the
// month's length, wraps ErrInvalidHijriDate.
func (c *UmmAlQura) ToCivilDate(d HijriDate, loc *time.Location) (time.Time, error) {
	if loc == nil {
		return time.Time{}, errors.New("timex: ToCivilDate requires a location")
	}
	if d.Month < 1 || d.Month > 12 {
		return time.Time{}, fmt.Errorf("%w: month %d is not 1 to 12", ErrInvalidHijriDate, d.Month)
	}
	i, ok := hijriMonthIndex(d.Year, d.Month)
	if !ok {
		return time.Time{}, fmt.Errorf("%w: %d H", ErrHijriOutOfRange, d.Year)
	}
	if n := c.starts[i+1] - c.starts[i]; d.Day < 1 || d.Day > n {
		return time.Time{}, fmt.Errorf("%w: day %d of month %d of %d H is not 1 to %d", ErrInvalidHijriDate, d.Day, d.Month, d.Year, n)
	}
	year, month, day := time.Unix(int64(c.starts[i]+d.Day-1)*secondsPerDay, 0).UTC().Date()
	return time.Date(year, month, day, 0, 0, 0, 0, loc), nil
}

// MonthLength returns the number of days in a Hijri month with offsets applied:
// 29 or 30, or 0 if the month is outside 1356-1500 H or month is not 1 to 12.
func (c *UmmAlQura) MonthLength(year, month int) int {
	i, ok := hijriMonthIndex(year, month)
	if !ok {
		return 0
	}
	return c.starts[i+1] - c.starts[i]
}

// hijriMonthShifts validates offsets and returns the shift for each month index.
func hijriMonthShifts(offsets []HijriMonthOffset) (map[int]int, error) {
	shifts := make(map[int]int, len(offsets))
	for _, o := range offsets {
		if o.Month < 1 || o.Month > 12 {
			return nil, fmt.Errorf("timex: Hijri month offset for %d H: month %d is not 1 to 12", o.Year, o.Month)
		}
		i, ok := hijriMonthIndex(o.Year, o.Month)
		if !ok {
			return nil, fmt.Errorf("%w: month offset for %d H", ErrHijriOutOfRange, o.Year)
		}
		if o.Days != -1 && o.Days != 1 {
			return nil, fmt.Errorf("timex: Hijri month offset for month %d of %d H: days %d is not -1 or 1", o.Month, o.Year, o.Days)
		}
		if _, dup := shifts[i]; dup {
			return nil, fmt.Errorf("timex: Hijri month offset for month %d of %d H given twice", o.Month, o.Year)
		}
		shifts[i] = o.Days
	}
	return shifts, nil
}

func hijriDateOutOfRange(year int, month time.Month, day int) error {
	return fmt.Errorf("%w: %04d-%02d-%02d", ErrHijriOutOfRange, year, int(month), day)
}

// hijriMonthIndex returns the table index of a month, counted from Muharram
// 1356 H.
func hijriMonthIndex(year, month int) (int, bool) {
	if year < ummAlQuraFirstYear || year > ummAlQuraLastYear || month < 1 || month > 12 {
		return 0, false
	}
	return (year-ummAlQuraFirstYear)*12 + month - 1, true
}

// hijriMonthAt is the inverse of hijriMonthIndex.
func hijriMonthAt(i int) (year, month int) {
	return ummAlQuraFirstYear + i/12, i%12 + 1
}

// ummAlQuraBaseLength returns the table length of the month at index i.
func ummAlQuraBaseLength(i int) int {
	if ummAlQuraMonthBits[i/12]>>(11-i%12)&1 == 1 {
		return 30
	}
	return 29
}
