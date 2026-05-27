// Package timex provides Indonesian-locale time helpers.
//
// Features:
//   - Bahasa Indonesia date formatting and parsing
//   - Indonesian timezone variables (Jakarta/Makassar/Jayapura)
//   - Indonesian public holiday calendar
//   - Business-day computation excluding weekends and holidays
//
// # Quick start
//
//	t := time.Date(2026, 5, 18, 0, 0, 0, 0, timex.Jakarta)
//	timex.FormatID(t, timex.LongID)   // "Senin, 18 Mei 2026"
//	timex.Age(birthday)                 // years since birthday in WIB
//	timex.BusinessDaysBetween(s, e)     // days excluding weekends + ID holidays
package timex
