package validx

import (
	"strings"

	"github.com/santekno/sdk/errx"
)

// NPWPInfo is the structured result of parsing an Indonesian Taxpayer ID
// (Nomor Pokok Wajib Pajak). NPWPs are 15 digits.
type NPWPInfo struct {
	Raw       string
	Formatted string // e.g., "01.234.567.8-901.000"
	Type      string // "Individual" or "Corporate"
}

// ParseNPWP validates and parses an NPWP string. The input may contain
// dots, dashes, or spaces — they are stripped before validation.
//
// On failure returns *errx.AppError with NPWP_INVALID_LENGTH or
// NPWP_NOT_NUMERIC.
func ParseNPWP(npwp string) (*NPWPInfo, error) {
	stripped := stripFormatting(npwp)
	if len(stripped) != 15 {
		return nil, errx.New("NPWP_INVALID_LENGTH",
			"NPWP must be exactly 15 digits when stripped").
			WithDetail("length", len(stripped))
	}
	for _, r := range stripped {
		if r < '0' || r > '9' {
			return nil, errx.New("NPWP_NOT_NUMERIC", "NPWP must contain only digits")
		}
	}

	typ := "Individual"
	if stripped[0] == '0' {
		typ = "Corporate"
	}

	return &NPWPInfo{
		Raw:       npwp,
		Formatted: FormatNPWP(stripped),
		Type:      typ,
	}, nil
}

// FormatNPWP formats a raw 15-digit NPWP string as "XX.XXX.XXX.X-XXX.XXX".
// Returns the original string unchanged if it cannot be formatted.
func FormatNPWP(raw string) string {
	stripped := stripFormatting(raw)
	if len(stripped) != 15 {
		return raw
	}
	return stripped[0:2] + "." +
		stripped[2:5] + "." +
		stripped[5:8] + "." +
		stripped[8:9] + "-" +
		stripped[9:12] + "." +
		stripped[12:15]
}

func stripFormatting(s string) string {
	r := strings.NewReplacer(".", "", "-", "", " ", "")
	return r.Replace(s)
}
