package timex

import (
	"errors"
	"strings"
	"time"

	idlocale "github.com/santekno/sdk/internal/locale/id"
)

// ErrInvalidFormat is returned by [ParseID] when the input cannot be parsed.
var ErrInvalidFormat = errors.New("timex: invalid format")

// FormatID formats t in Bahasa Indonesia using the supplied layout.
// Use one of the package-level layout constants (LongID, ShortID, DateID,
// DateTimeID, TimeID), or any standard Go reference layout — English month
// and weekday names in the output are replaced with Bahasa Indonesia.
//
// Returns the empty string if t is the zero value.
func FormatID(t time.Time, layout string) string {
	if t.IsZero() {
		return ""
	}
	out := t.Format(layout)

	// Replace full names first to avoid prefix collisions with short forms.
	out = strings.ReplaceAll(out, t.Weekday().String(), idlocale.Weekdays[int(t.Weekday())])
	out = strings.ReplaceAll(out, t.Month().String(), idlocale.Months[int(t.Month())])

	// Then short forms.
	out = strings.ReplaceAll(out, t.Weekday().String()[:3], idlocale.ShortWeekdays[int(t.Weekday())])
	monthEN := t.Month().String()
	if len(monthEN) >= 3 {
		out = strings.ReplaceAll(out, monthEN[:3], idlocale.ShortMonths[int(t.Month())])
	}

	return out
}

// ParseID parses a Bahasa Indonesia formatted date string in the Asia/Jakarta
// timezone. Indonesian month/weekday tokens in value are substituted back to
// their English equivalents before delegating to time.ParseInLocation.
func ParseID(layout, value string) (time.Time, error) {
	// Substitute full names first.
	for i := 1; i <= 12; i++ {
		value = strings.ReplaceAll(value, idlocale.Months[i], time.Month(i).String())
	}
	for i := 0; i < 7; i++ {
		value = strings.ReplaceAll(value, idlocale.Weekdays[i], time.Weekday(i).String())
	}
	// Then short forms.
	for i := 1; i <= 12; i++ {
		value = strings.ReplaceAll(value, idlocale.ShortMonths[i], time.Month(i).String()[:3])
	}
	for i := 0; i < 7; i++ {
		value = strings.ReplaceAll(value, idlocale.ShortWeekdays[i], time.Weekday(i).String()[:3])
	}

	t, err := time.ParseInLocation(layout, value, Jakarta)
	if err != nil {
		return time.Time{}, ErrInvalidFormat
	}
	return t, nil
}
