package validx

import (
	"fmt"
	"strconv"
)

// IDROption customizes [FormatIDR] output.
type IDROption func(*idrConfig)

type idrConfig struct {
	withDecimal bool
}

// WithDecimal includes ",00" after the integer portion.
func WithDecimal(b bool) IDROption {
	return func(c *idrConfig) { c.withDecimal = b }
}

// FormatIDR formats an integer amount as Indonesian Rupiah.
//
//	FormatIDR(1500000)                       // "Rp1.500.000"
//	FormatIDR(1500000, WithDecimal(true))    // "Rp1.500.000,00"
func FormatIDR(amount int64, opts ...IDROption) string {
	cfg := &idrConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	neg := amount < 0
	if neg {
		amount = -amount
	}
	s := strconv.FormatInt(amount, 10)
	// insert '.' as thousands separator
	n := len(s)
	if n <= 3 {
		out := "Rp" + s
		if neg {
			out = "-" + out
		}
		if cfg.withDecimal {
			out += ",00"
		}
		return out
	}
	// build right-to-left
	buf := make([]byte, 0, n+(n/3))
	for i, c := range []byte(s) {
		rem := n - i
		if i > 0 && rem%3 == 0 {
			buf = append(buf, '.')
		}
		buf = append(buf, c)
	}
	out := fmt.Sprintf("Rp%s", string(buf))
	if neg {
		out = "-" + out
	}
	if cfg.withDecimal {
		out += ",00"
	}
	return out
}
