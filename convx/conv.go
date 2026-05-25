// Package convx provides safe type conversion helpers.
//
//	n := convx.ToInt("42")          // 42, true
//	s := convx.ToString(3.14)       // "3.14"
//	v := convx.Must(strconv.Atoi("5")) // 5 (panics on error)
package convx

import (
	"fmt"
	"strconv"
)

// ToInt converts v to int. Returns (0, false) if conversion fails.
func ToInt(v any) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int8:
		return int(x), true
	case int16:
		return int(x), true
	case int32:
		return int(x), true
	case int64:
		return int(x), true
	case uint:
		return int(x), true
	case uint8:
		return int(x), true
	case uint16:
		return int(x), true
	case uint32:
		return int(x), true
	case uint64:
		if x > uint64(^uint(0)>>1) { //nolint:gosec // intentional: caller handles false
			return 0, false
		}
		return int(x), true //nolint:gosec // bounds-checked above
	case float32:
		return int(x), true
	case float64:
		return int(x), true
	case string:
		n, err := strconv.Atoi(x)
		return n, err == nil
	case bool:
		if x {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

// ToString converts v to its string representation using fmt.Sprint.
func ToString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

// ToBool converts v to bool. Returns (false, false) if conversion fails.
func ToBool(v any) (bool, bool) {
	switch x := v.(type) {
	case bool:
		return x, true
	case string:
		b, err := strconv.ParseBool(x)
		return b, err == nil
	case int:
		return x != 0, true
	case int64:
		return x != 0, true
	case float64:
		return x != 0, true
	}
	return false, false
}

// ToFloat64 converts v to float64. Returns (0, false) if conversion fails.
func ToFloat64(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case string:
		f, err := strconv.ParseFloat(x, 64)
		return f, err == nil
	}
	return 0, false
}

// Must returns v if err is nil, otherwise it panics with err.
// Useful for wrapping functions that return (T, error) in init-time code.
func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
