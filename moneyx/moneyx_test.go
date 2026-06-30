package moneyx

import "testing"

func TestExactArithmetic(t *testing.T) {
	// 15,700,000 / 12 stays exact; multiplied back equals the original.
	total := FromInt(15_700_000)
	avg := total.Div(FromInt(12))
	if got := avg.StringFixed(2); got != "1308333.33" {
		t.Errorf("avg = %s, want 1308333.33", got)
	}
	if back := avg.Mul(FromInt(12)); back.Cmp(total) != 0 {
		t.Errorf("avg*12 = %s, want 15700000 exactly (no float drift)", back.String())
	}
}

func TestCmpSignZero(t *testing.T) {
	if !Zero().IsZero() {
		t.Error("Zero should be zero")
	}
	if FromInt(-5).Sign() != -1 {
		t.Error("-5 sign should be -1")
	}
	if FromInt(3).Cmp(FromInt(5)) != -1 {
		t.Error("3 < 5")
	}
}

func TestFormatIDR(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "Rp0"},
		{1000, "Rp1.000"},
		{1350000, "Rp1.350.000"},
		{248500000, "Rp248.500.000"},
		{-2700000, "-Rp2.700.000"},
	}
	for _, c := range cases {
		if got := FormatIDR(c.in); got != c.want {
			t.Errorf("FormatIDR(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFormatIDRDec(t *testing.T) {
	if got := FormatIDRDec(MustParse("1350000.4")); got != "Rp1.350.000" {
		t.Errorf("got %s, want Rp1.350.000", got)
	}
}
