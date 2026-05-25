package timex

// Layout constants for Bahasa Indonesia date formatting.
//
// These use Go's standard reference-time tokens (Monday, January, 2006…) so
// time.Format works correctly; FormatID then substitutes English month and
// weekday names with their Bahasa Indonesia equivalents.
const (
	// LongID → e.g. "Senin, 18 Mei 2026"
	LongID = "Monday, 02 January 2006"
	// ShortID → e.g. "18 Mei 2026"
	ShortID = "02 Jan 2006"
	// DateID → e.g. "18 Mei 2026"
	DateID = "02 January 2006"
	// DateTimeID → e.g. "Senin, 18 Mei 2026 09:30 WIB"
	DateTimeID = "Monday, 02 January 2006 15:04 WIB"
	// TimeID → e.g. "09:30 WIB"
	TimeID = "15:04 WIB"
)
