// Package numx provides generic numeric helper utilities.
package numx

import "math"

// Number is the constraint for numeric generics in numx.
type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// Min returns the smaller of a and b.
func Min[T Number](a, b T) T {
	if a < b {
		return a
	}
	return b
}

// Max returns the larger of a and b.
func Max[T Number](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// Clamp constrains v to the range [lo, hi].
func Clamp[T Number](v, lo, hi T) T {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Abs returns the absolute value of v.
func Abs[T Number](v T) T {
	if v < 0 {
		return -v
	}
	return v
}

// Sum returns the sum of all elements in s.
func Sum[T Number](s []T) T {
	var total T
	for _, v := range s {
		total += v
	}
	return total
}

// Average returns the arithmetic mean of s. Returns 0 for empty slices.
func Average[T Number](s []T) float64 {
	if len(s) == 0 {
		return 0
	}
	return float64(Sum(s)) / float64(len(s))
}

// Round rounds f to the nearest integer using round-half-away-from-zero.
func Round(f float64) float64 { return math.Round(f) }

// RoundToDecimal rounds f to the given number of decimal places.
func RoundToDecimal(f float64, decimals int) float64 {
	shift := math.Pow(10, float64(decimals))
	return math.Round(f*shift) / shift
}
