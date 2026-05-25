package timex

import "time"

// Age returns the number of complete years between birthday and the current
// time in the Asia/Jakarta timezone. Returns 0 if birthday is in the future.
func Age(birthday time.Time) int {
	now := time.Now().In(Jakarta)
	b := birthday.In(Jakarta)
	years := now.Year() - b.Year()
	if now.Month() < b.Month() || (now.Month() == b.Month() && now.Day() < b.Day()) {
		years--
	}
	if years < 0 {
		return 0
	}
	return years
}

// IsBusinessDay reports whether t is a Monday–Friday non-holiday in the
// Asia/Jakarta timezone.
func IsBusinessDay(t time.Time) bool {
	t = t.In(Jakarta)
	wd := t.Weekday()
	if wd == time.Saturday || wd == time.Sunday {
		return false
	}
	return !IsIndonesianHoliday(t)
}

// BusinessDaysBetween returns the number of business days between start and
// end (exclusive of end), excluding weekends and Indonesian public holidays.
// Returns 0 if end is on or before start.
func BusinessDaysBetween(start, end time.Time) int {
	start = startOfDay(start.In(Jakarta))
	end = startOfDay(end.In(Jakarta))
	if !end.After(start) {
		return 0
	}
	count := 0
	for d := start; d.Before(end); d = d.AddDate(0, 0, 1) {
		if IsBusinessDay(d) {
			count++
		}
	}
	return count
}

// AddBusinessDays returns the date that is n business days after t.
// If n is negative, returns t unchanged (negative is not supported).
func AddBusinessDays(t time.Time, n int) time.Time {
	if n < 0 {
		return t
	}
	d := t
	for added := 0; added < n; {
		d = d.AddDate(0, 0, 1)
		if IsBusinessDay(d) {
			added++
		}
	}
	return d
}

func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
