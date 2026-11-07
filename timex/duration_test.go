package timex_test

import (
	"errors"
	"testing"
	"time"

	"github.com/santekno/sdk/timex"
)

func TestParseISODuration(t *testing.T) {
	tests := []struct {
		in   string
		want time.Duration
	}{
		{"PT1S", time.Second},
		{"PT0S", 0},
		{"P0D", 0},
		{"PT1M", time.Minute},
		{"PT1H", time.Hour},
		{"PT1H2M3S", time.Hour + 2*time.Minute + 3*time.Second},
		{"PT59M59S", 59*time.Minute + 59*time.Second},
		{"PT90M", 90 * time.Minute},
		{"P1D", 24 * time.Hour},
		{"P1DT2H", 26 * time.Hour},
		{"P2DT3H4M5S", 48*time.Hour + 3*time.Hour + 4*time.Minute + 5*time.Second},
		{"PT0.5S", 500 * time.Millisecond},
		{"PT1.25S", 1250 * time.Millisecond},
		{"PT0.000000001S", time.Nanosecond},
		{"PT1H0.5S", time.Hour + 500*time.Millisecond},
		{"PT007S", 7 * time.Second},
		{"PT2562047H47M16.854775807S", time.Duration(1<<63 - 1)},
		{"P106751DT23H47M16.854775807S", time.Duration(1<<63 - 1)},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := timex.ParseISODuration(tt.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseISODuration_Invalid(t *testing.T) {
	tests := []struct{ name, in string }{
		{"empty", ""},
		{"no P", "T1S"},
		{"lowercase", "pt1s"},
		{"lowercase designator", "PT1h"},
		{"P only", "P"},
		{"PT only", "PT"},
		{"trailing T", "P1DT"},
		{"duplicate T", "PTT1S"},
		{"T after time", "PT1HT1S"},
		{"years", "P1Y"},
		{"months", "P1M"},
		{"months with time", "P1MT1S"},
		{"years and time", "P1YT1H"},
		{"weeks", "P1W"},
		{"negative", "-PT1S"},
		{"negative number", "PT-1S"},
		{"plus sign", "PT+1S"},
		{"fraction on minutes", "PT1.5M"},
		{"fraction on days", "P1.5D"},
		{"comma fraction", "PT1,5S"},
		{"empty fraction", "PT1.S"},
		{"no leading digit", "PT.5S"},
		{"fraction too long", "PT0.0000000001S"},
		{"no designator", "PT1"},
		{"no digits", "PTS"},
		{"wrong order", "PT1S1M"},
		{"repeated", "PT1M1M"},
		{"D after T", "PT1D"},
		{"H before T", "P1H"},
		{"S before T", "P1S"},
		{"unknown designator", "PT1X"},
		{"space", "PT1S "},
		{"leading space", " PT1S"},
		{"non-ASCII digit", "PT١S"},
		{"overflow hours", "PT2562048H"},
		{"overflow sum", "PT2562047H48M"},
		{"overflow by one ns", "PT9223372036.854775808S"},
		{"overflow days", "P106752D"},
		{"overflow huge digits", "PT99999999999999999999999999S"},
		{"overflow days sum", "P106751DT24H"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := timex.ParseISODuration(tt.in)
			if err == nil {
				t.Fatalf("ParseISODuration(%q) = %v, want error", tt.in, got)
			}
			if !errors.Is(err, timex.ErrInvalidISODuration) {
				t.Fatalf("error %v does not wrap ErrInvalidISODuration", err)
			}
			if got != 0 {
				t.Fatalf("got %v on error, want 0", got)
			}
		})
	}
}
