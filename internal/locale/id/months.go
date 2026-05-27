// Package id provides Indonesian locale data as a fallback when
// github.com/goodsign/monday is unavailable.
package id

// Months maps Go time.Month (1–12) to the Bahasa Indonesia month name.
var Months = map[int]string{
	1:  "Januari",
	2:  "Februari",
	3:  "Maret",
	4:  "April",
	5:  "Mei",
	6:  "Juni",
	7:  "Juli",
	8:  "Agustus",
	9:  "September",
	10: "Oktober",
	11: "November",
	12: "Desember",
}

// ShortMonths maps Go time.Month (1–12) to the short Bahasa Indonesia month name.
var ShortMonths = map[int]string{
	1:  "Jan",
	2:  "Feb",
	3:  "Mar",
	4:  "Apr",
	5:  "Mei",
	6:  "Jun",
	7:  "Jul",
	8:  "Agu",
	9:  "Sep",
	10: "Okt",
	11: "Nov",
	12: "Des",
}
