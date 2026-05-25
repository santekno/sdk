package validx_test

import (
	"testing"

	"github.com/santekno/sdk/validx"
)

func FuzzParseNIK(f *testing.F) {
	f.Add("3201010101800001")
	f.Add("0000000000000000")
	f.Add("")
	f.Add("320101010180000X")
	f.Fuzz(func(t *testing.T, s string) {
		_, _ = validx.ParseNIK(s)
	})
}

func FuzzParseNPWP(f *testing.F) {
	f.Add("012345678901000")
	f.Add("01.234.567.8-901.000")
	f.Add("")
	f.Fuzz(func(t *testing.T, s string) {
		_, _ = validx.ParseNPWP(s)
	})
}
