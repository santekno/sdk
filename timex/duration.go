package timex

import (
	"errors"
	"fmt"
	"math"
	"time"
)

// ErrInvalidISODuration is returned by ParseISODuration for input that is not
// in the supported PnDTnHnMnS subset, and for values that overflow
// time.Duration. Use errors.Is to test for it.
var ErrInvalidISODuration = errors.New("timex: invalid ISO 8601 duration")

// ParseISODuration parses an ISO 8601 duration such as the contentDetails.duration
// of a YouTube video ("PT1H2M3S", "P1DT2H", "P0D") into a time.Duration.
//
// Accepted grammar: "P" [nD] ["T" [nH] [nM] [nS]], with designators in that
// order, each at most once, and at least one component. Rules:
//   - designators are upper case only; "pt1s" is rejected
//   - years, months and weeks are rejected because their length is not fixed
//   - a day is exactly 24 hours (no daylight-saving or leap handling)
//   - the number is unsigned ASCII digits; a leading sign is rejected, so
//     negative durations are not supported
//   - only the seconds component may carry a fraction, with "." as separator
//     and at most 9 digits ("PT0.5S", "PT1.25S"); "," is rejected
//   - "T" must be followed by at least one time component ("P1DT" is rejected)
//   - empty input, "P" and "PT" are rejected
//   - a total that does not fit in time.Duration (about 292 years) is rejected
//
// All failures wrap ErrInvalidISODuration.
func ParseISODuration(s string) (time.Duration, error) {
	fail := func(why string) (time.Duration, error) {
		return 0, fmt.Errorf("%w: %s", ErrInvalidISODuration, why)
	}
	if s == "" {
		return fail("empty")
	}
	if s[0] != 'P' {
		return fail("missing leading P")
	}
	rest := s[1:]
	var (
		total      uint64
		components int
		inTime     bool
		lastRank   int // 1=D 2=H 3=M 4=S
	)
	const maxDur = uint64(math.MaxInt64)

	add := func(v uint64, unit time.Duration) bool {
		if v > maxDur/uint64(unit) {
			return false
		}
		n := v * uint64(unit)
		if total > maxDur-n {
			return false
		}
		total += n
		return true
	}

	for len(rest) > 0 {
		if rest[0] == 'T' {
			if inTime {
				return fail("duplicate T")
			}
			inTime = true
			rest = rest[1:]
			if len(rest) == 0 {
				return fail("T without time component")
			}
			continue
		}
		// Integer part.
		i := 0
		var whole uint64
		for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' {
			d := uint64(rest[i] - '0')
			if whole > (maxDur-d)/10 {
				return fail("overflow")
			}
			whole = whole*10 + d
			i++
		}
		if i == 0 {
			return fail("expected digits")
		}
		var frac uint64 // nanoseconds
		hasFrac := false
		if i < len(rest) && rest[i] == '.' {
			hasFrac = true
			i++
			start := i
			scale := uint64(100_000_000)
			for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' {
				if i-start >= 9 {
					return fail("fraction longer than 9 digits")
				}
				frac += uint64(rest[i]-'0') * scale
				scale /= 10
				i++
			}
			if i == start {
				return fail("empty fraction")
			}
		}
		if i >= len(rest) {
			return fail("number without designator")
		}
		des := rest[i]
		rest = rest[i+1:]

		var rank int
		var unit time.Duration
		switch {
		case des == 'D' && !inTime:
			rank, unit = 1, 24*time.Hour
		case des == 'H' && inTime:
			rank, unit = 2, time.Hour
		case des == 'M' && inTime:
			rank, unit = 3, time.Minute
		case des == 'S' && inTime:
			rank, unit = 4, time.Second
		case des == 'Y' || des == 'W' || (des == 'M' && !inTime):
			return fail("years, months and weeks are not supported")
		default:
			return fail(fmt.Sprintf("unexpected designator %q", des))
		}
		if rank <= lastRank {
			return fail("designators out of order or repeated")
		}
		lastRank = rank
		if hasFrac && rank != 4 {
			return fail("fraction allowed on seconds only")
		}
		if !add(whole, unit) || !add(frac, 1) {
			return fail("overflow")
		}
		components++
	}
	if components == 0 {
		return fail("no components")
	}
	return time.Duration(total), nil
}
