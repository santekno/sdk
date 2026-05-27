package validx_test

import (
	"errors"
	"testing"

	"github.com/santekno/sdk/errx"
	"github.com/santekno/sdk/validx"
)

func TestParseNIK_Valid(t *testing.T) {
	// Province 32 = Jawa Barat. Day=15 (male), Month=05, Year=90 → 1990-05-15.
	info, err := validx.ParseNIK("3201151505900123")
	if err != nil {
		t.Fatalf("ParseNIK error: %v", err)
	}
	if info.Province != "Jawa Barat" {
		t.Errorf("Province = %q, want Jawa Barat", info.Province)
	}
	if info.Gender != "M" {
		t.Errorf("Gender = %q, want M", info.Gender)
	}
	if info.BirthDate.Year() != 1990 || info.BirthDate.Month() != 5 || info.BirthDate.Day() != 15 {
		t.Errorf("BirthDate = %v, want 1990-05-15", info.BirthDate)
	}
	if info.SerialNumber != "0123" {
		t.Errorf("SerialNumber = %q, want 0123", info.SerialNumber)
	}
}

func TestParseNIK_Female(t *testing.T) {
	// Day 55 = 15 + 40 → female, day 15.
	info, err := validx.ParseNIK("3201155505900123")
	if err != nil {
		t.Fatalf("ParseNIK error: %v", err)
	}
	if info.Gender != "F" {
		t.Errorf("Gender = %q, want F", info.Gender)
	}
	if info.BirthDate.Day() != 15 {
		t.Errorf("Day = %d, want 15", info.BirthDate.Day())
	}
}

func TestParseNIK_InvalidLength(t *testing.T) {
	_, err := validx.ParseNIK("1234")
	if err == nil {
		t.Fatal("expected error")
	}
	if errx.Code(err) != "NIK_INVALID_LENGTH" {
		t.Errorf("Code = %q, want NIK_INVALID_LENGTH", errx.Code(err))
	}
}

func TestParseNIK_NotNumeric(t *testing.T) {
	_, err := validx.ParseNIK("32011515X5900123")
	if errx.Code(err) != "NIK_NOT_NUMERIC" {
		t.Errorf("Code = %q, want NIK_NOT_NUMERIC", errx.Code(err))
	}
}

func TestParseNIK_InvalidProvince(t *testing.T) {
	_, err := validx.ParseNIK("9901151505900123")
	if errx.Code(err) != "NIK_INVALID_PROVINCE" {
		t.Errorf("Code = %q, want NIK_INVALID_PROVINCE", errx.Code(err))
	}
}

func TestParseNIK_InvalidDate(t *testing.T) {
	// Month 99 — invalid
	_, err := validx.ParseNIK("3201159905900123")
	if errx.Code(err) != "NIK_INVALID_DATE" {
		t.Errorf("Code = %q, want NIK_INVALID_DATE", errx.Code(err))
	}
}

func TestParseNIK_AppErrorChain(t *testing.T) {
	_, err := validx.ParseNIK("xx")
	var ae *errx.AppError
	if !errors.As(err, &ae) {
		t.Fatal("expected *errx.AppError")
	}
}

func TestParseNPWP_Valid(t *testing.T) {
	info, err := validx.ParseNPWP("012345678901000")
	if err != nil {
		t.Fatalf("ParseNPWP error: %v", err)
	}
	if info.Type != "Corporate" {
		t.Errorf("Type = %q, want Corporate", info.Type)
	}
	if info.Formatted != "01.234.567.8-901.000" {
		t.Errorf("Formatted = %q", info.Formatted)
	}
}

func TestParseNPWP_Individual(t *testing.T) {
	info, err := validx.ParseNPWP("123456789012345")
	if err != nil {
		t.Fatalf("ParseNPWP error: %v", err)
	}
	if info.Type != "Individual" {
		t.Errorf("Type = %q, want Individual", info.Type)
	}
}

func TestParseNPWP_StripsFormatting(t *testing.T) {
	info, err := validx.ParseNPWP("01.234.567.8-901.000")
	if err != nil {
		t.Fatalf("ParseNPWP error: %v", err)
	}
	if info.Formatted != "01.234.567.8-901.000" {
		t.Errorf("Formatted = %q", info.Formatted)
	}
}

func TestParseNPWP_InvalidLength(t *testing.T) {
	_, err := validx.ParseNPWP("123")
	if errx.Code(err) != "NPWP_INVALID_LENGTH" {
		t.Errorf("Code = %q, want NPWP_INVALID_LENGTH", errx.Code(err))
	}
}

func TestFormatNPWP_Passthrough(t *testing.T) {
	if got := validx.FormatNPWP("bad"); got != "bad" {
		t.Errorf("FormatNPWP(bad) = %q, want bad", got)
	}
}

func TestFormatIDR(t *testing.T) {
	cases := []struct {
		in   int64
		opts []validx.IDROption
		want string
	}{
		{1500000, nil, "Rp1.500.000"},
		{500, nil, "Rp500"},
		{0, nil, "Rp0"},
		{1500000, []validx.IDROption{validx.WithDecimal(true)}, "Rp1.500.000,00"},
		{-1000, nil, "-Rp1.000"},
		{1000000000, nil, "Rp1.000.000.000"},
	}
	for _, c := range cases {
		got := validx.FormatIDR(c.in, c.opts...)
		if got != c.want {
			t.Errorf("FormatIDR(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}
