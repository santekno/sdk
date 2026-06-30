// Package moneyx provides an exact decimal money type and Indonesian Rupiah
// formatting. It is backed by math/big.Rat so all arithmetic is exact — never
// float — which is required for financial correctness.
//
//	a := moneyx.MustParse("1350000")
//	b := moneyx.FromInt(3)
//	moneyx.FormatIDRDec(a.Mul(b)) // "Rp4.050.000"
package moneyx

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

// Dec is an exact decimal amount (money, quantity, or price), backed by big.Rat.
// The zero value is a valid 0. Dec is immutable: every operation returns a new Dec.
type Dec struct{ r *big.Rat }

func (d Dec) rat() *big.Rat {
	if d.r == nil {
		return new(big.Rat)
	}
	return d.r
}

// Zero returns 0.
func Zero() Dec { return Dec{new(big.Rat)} }

// FromInt returns an exact Dec from an integer.
func FromInt(i int64) Dec { return Dec{new(big.Rat).SetInt64(i)} }

// Parse reads a decimal string such as "1350000" or "1308.33".
func Parse(s string) (Dec, error) {
	r, ok := new(big.Rat).SetString(strings.TrimSpace(s))
	if !ok {
		return Dec{}, fmt.Errorf("moneyx: invalid decimal %q", s)
	}
	return Dec{r}, nil
}

// MustParse is Parse that panics on error (for tests and constant literals).
func MustParse(s string) Dec {
	d, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return d
}

// Add returns d + o.
func (d Dec) Add(o Dec) Dec { return Dec{new(big.Rat).Add(d.rat(), o.rat())} }

// Sub returns d - o.
func (d Dec) Sub(o Dec) Dec { return Dec{new(big.Rat).Sub(d.rat(), o.rat())} }

// Mul returns d * o.
func (d Dec) Mul(o Dec) Dec { return Dec{new(big.Rat).Mul(d.rat(), o.rat())} }

// Div returns d / o. The caller must ensure o is non-zero.
func (d Dec) Div(o Dec) Dec { return Dec{new(big.Rat).Quo(d.rat(), o.rat())} }

// Neg returns -d.
func (d Dec) Neg() Dec { return Dec{new(big.Rat).Neg(d.rat())} }

// Cmp returns -1, 0, or +1 as d is less than, equal to, or greater than o.
func (d Dec) Cmp(o Dec) int { return d.rat().Cmp(o.rat()) }

// Sign returns -1, 0, or +1.
func (d Dec) Sign() int { return d.rat().Sign() }

// IsZero reports whether d == 0.
func (d Dec) IsZero() bool { return d.rat().Sign() == 0 }

// StringFixed renders d with exactly prec decimal places (rounded).
func (d Dec) StringFixed(prec int) string { return d.rat().FloatString(prec) }

// String renders d with trailing zeros trimmed (up to 8 decimal places).
func (d Dec) String() string {
	s := d.rat().FloatString(8)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	return s
}

// RoundIDR rounds d to the nearest whole Rupiah.
func (d Dec) RoundIDR() int64 {
	var i big.Int
	i.SetString(d.rat().FloatString(0), 10)
	return i.Int64()
}

// FormatIDR formats a whole-Rupiah amount with Indonesian thousand separators,
// e.g. 1350000 -> "Rp1.350.000", -2700000 -> "-Rp2.700.000".
func FormatIDR(rupiah int64) string {
	neg := rupiah < 0
	if neg {
		rupiah = -rupiah
	}
	digits := strconv.FormatInt(rupiah, 10)
	n := len(digits)
	var b []byte
	for i := 0; i < n; i++ {
		if i > 0 && (n-i)%3 == 0 {
			b = append(b, '.')
		}
		b = append(b, digits[i])
	}
	out := "Rp" + string(b)
	if neg {
		out = "-" + out
	}
	return out
}

// FormatIDRDec formats a Dec as Rupiah (rounded), e.g. "Rp1.350.000".
func FormatIDRDec(d Dec) string { return FormatIDR(d.RoundIDR()) }
