package timex

import "time"

// HolidayType classifies a public holiday.
type HolidayType int

// HolidayType values.
const (
	HolidayNational HolidayType = iota
	HolidayReligious
	HolidayRegional
)

// Holiday represents a single entry in the Indonesian public holiday calendar.
type Holiday struct {
	Date time.Time
	Name string // Bahasa Indonesia name
	Type HolidayType
}

// indonesianHolidays is the bundled holiday calendar, updated annually.
// Source: official Indonesian government Keputusan Presiden / Peraturan Presiden.
var indonesianHolidays = map[int][]Holiday{
	2026: {
		{Date: date(2026, 1, 1), Name: "Tahun Baru Masehi", Type: HolidayNational},
		{Date: date(2026, 1, 17), Name: "Isra Mikraj Nabi Muhammad SAW", Type: HolidayReligious},
		{Date: date(2026, 2, 17), Name: "Tahun Baru Imlek", Type: HolidayReligious},
		{Date: date(2026, 3, 20), Name: "Hari Suci Nyepi", Type: HolidayReligious},
		{Date: date(2026, 3, 21), Name: "Idul Fitri 1447 H (Hari Pertama)", Type: HolidayReligious},
		{Date: date(2026, 3, 22), Name: "Idul Fitri 1447 H (Hari Kedua)", Type: HolidayReligious},
		{Date: date(2026, 4, 3), Name: "Wafat Isa Almasih", Type: HolidayReligious},
		{Date: date(2026, 5, 1), Name: "Hari Buruh Internasional", Type: HolidayNational},
		{Date: date(2026, 5, 14), Name: "Kenaikan Isa Almasih", Type: HolidayReligious},
		{Date: date(2026, 5, 27), Name: "Hari Raya Waisak", Type: HolidayReligious},
		{Date: date(2026, 5, 28), Name: "Idul Adha 1447 H", Type: HolidayReligious},
		{Date: date(2026, 6, 1), Name: "Hari Lahir Pancasila", Type: HolidayNational},
		{Date: date(2026, 6, 17), Name: "Tahun Baru Hijriyah 1448 H", Type: HolidayReligious},
		{Date: date(2026, 8, 17), Name: "Hari Kemerdekaan Republik Indonesia", Type: HolidayNational},
		{Date: date(2026, 8, 26), Name: "Maulid Nabi Muhammad SAW", Type: HolidayReligious},
		{Date: date(2026, 12, 25), Name: "Hari Raya Natal", Type: HolidayReligious},
	},
	2027: {
		{Date: date(2027, 1, 1), Name: "Tahun Baru Masehi", Type: HolidayNational},
		{Date: date(2027, 5, 1), Name: "Hari Buruh Internasional", Type: HolidayNational},
		{Date: date(2027, 6, 1), Name: "Hari Lahir Pancasila", Type: HolidayNational},
		{Date: date(2027, 8, 17), Name: "Hari Kemerdekaan Republik Indonesia", Type: HolidayNational},
		{Date: date(2027, 12, 25), Name: "Hari Raya Natal", Type: HolidayReligious},
	},
}

func date(y, m, d int) time.Time {
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, Jakarta)
}

// IndonesianHolidays returns all public holidays bundled for the given year.
// Returns an empty slice for years outside the bundled range.
func IndonesianHolidays(year int) []Holiday {
	if h, ok := indonesianHolidays[year]; ok {
		out := make([]Holiday, len(h))
		copy(out, h)
		return out
	}
	return []Holiday{}
}

// IsIndonesianHoliday reports whether t falls on a bundled Indonesian holiday.
// Comparison is done by Year-Month-Day in the Asia/Jakarta timezone.
func IsIndonesianHoliday(t time.Time) bool {
	t = t.In(Jakarta)
	for _, h := range IndonesianHolidays(t.Year()) {
		if h.Date.Year() == t.Year() && h.Date.Month() == t.Month() && h.Date.Day() == t.Day() {
			return true
		}
	}
	return false
}
