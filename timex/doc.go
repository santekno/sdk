// Package timex provides Indonesian-locale time helpers.
//
// Features:
//   - Bahasa Indonesia date formatting and parsing
//   - Indonesian timezone variables (Jakarta/Makassar/Jayapura)
//   - Indonesian public holiday calendar
//   - Business-day computation excluding weekends and holidays
//   - Gregorian to Hijri conversion with the Umm al-Qura calendar
//
// # Quick start
//
//	t := time.Date(2026, 5, 18, 0, 0, 0, 0, timex.Jakarta)
//	timex.FormatID(t, timex.LongID)   // "Senin, 18 Mei 2026"
//	timex.Age(birthday)                 // years since birthday in WIB
//	timex.BusinessDaysBetween(s, e)     // days excluding weekends + ID holidays
//
// # Hijri dates
//
// NewUmmAlQura builds a calendar from an embedded table covering 1356-1500 H.
// Conversion takes the civil date of an instant in an explicit location, so the
// Hijri day changes at local midnight rather than at sunset. Dates outside the
// table return ErrHijriOutOfRange. Local month sightings are applied with
// WithHijriMonthOffsets, which rejects offsets that leave a month with other
// than 29 or 30 days.
//
//	cal, err := timex.NewUmmAlQura(timex.WithHijriMonthOffsets(
//		timex.HijriMonthOffset{Year: 1446, Month: 10, Days: 1},
//	))
//	h, err := cal.FromCivilDate(t, timex.Jakarta) // HijriDate{Year, Month, Day}
//	n := cal.MonthLength(1448, 5)                 // 29 or 30, 0 outside the table
//
// The table is the one in ICU, the data behind the JavaScript Intl calendar
// "islamic-umalqura". It matches R.H. van Gent's reconstruction of the Saudi
// calendar for 1420-1450 H and differs from it outside those years; see
// hijri_data.go.
package timex
