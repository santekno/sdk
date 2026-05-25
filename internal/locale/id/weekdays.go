package id

// Weekdays maps Go time.Weekday (0=Sunday) to the Bahasa Indonesia weekday name.
var Weekdays = map[int]string{
	0: "Minggu",
	1: "Senin",
	2: "Selasa",
	3: "Rabu",
	4: "Kamis",
	5: "Jumat",
	6: "Sabtu",
}

// ShortWeekdays maps Go time.Weekday (0=Sunday) to the short Bahasa Indonesia weekday name.
var ShortWeekdays = map[int]string{
	0: "Min",
	1: "Sen",
	2: "Sel",
	3: "Rab",
	4: "Kam",
	5: "Jum",
	6: "Sab",
}
