package timex

import "time"

// Indonesian timezone variables.
//
//	Jakarta  — Asia/Jakarta  (WIB, UTC+7)
//	Makassar — Asia/Makassar (WITA, UTC+8)
//	Jayapura — Asia/Jayapura (WIT, UTC+9)
//
// Initialized from the system tzdata at package init. If LoadLocation fails
// (missing tzdata), a fixed-offset fallback is used.
var (
	Jakarta  = mustLoadLocation("Asia/Jakarta", 7)
	Makassar = mustLoadLocation("Asia/Makassar", 8)
	Jayapura = mustLoadLocation("Asia/Jayapura", 9)
)

func mustLoadLocation(name string, offsetHours int) *time.Location {
	loc, err := time.LoadLocation(name)
	if err == nil {
		return loc
	}
	return time.FixedZone(name, offsetHours*3600)
}

// ToJakarta returns t in the Asia/Jakarta timezone (WIB, UTC+7).
func ToJakarta(t time.Time) time.Time { return t.In(Jakarta) }

// ToWITA returns t in the Asia/Makassar timezone (WITA, UTC+8).
func ToWITA(t time.Time) time.Time { return t.In(Makassar) }

// ToWIT returns t in the Asia/Jayapura timezone (WIT, UTC+9).
func ToWIT(t time.Time) time.Time { return t.In(Jayapura) }
