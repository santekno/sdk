package timex_test

import (
	"errors"
	"fmt"
	"time"

	"github.com/santekno/sdk/timex"
)

func ExampleNewUmmAlQura() {
	cal, err := timex.NewUmmAlQura()
	if err != nil {
		panic(err)
	}

	// 05.00 WIB on 29 October 2026 is still 28 October in UTC.
	at := time.Date(2026, 10, 28, 22, 0, 0, 0, time.UTC)
	h, err := cal.FromCivilDate(at, timex.Jakarta)
	if err != nil {
		panic(err)
	}
	fmt.Println(h.Day, h.Month, h.Year)
	// Output: 18 5 1448
}

func ExampleWithHijriMonthOffsets() {
	// Kementerian Agama fixed 1 Syawal 1446 H a day after the Umm al-Qura table.
	cal, err := timex.NewUmmAlQura(timex.WithHijriMonthOffsets(
		timex.HijriMonthOffset{Year: 1446, Month: 10, Days: 1},
	))
	if err != nil {
		panic(err)
	}

	at := time.Date(2025, 3, 30, 12, 0, 0, 0, timex.Jakarta)
	h, err := cal.FromCivilDate(at, timex.Jakarta)
	if err != nil {
		panic(err)
	}
	fmt.Println(h.Day, h.Month, h.Year)
	// Output: 30 9 1446
}

func ExampleUmmAlQura_FromCivilDate_outOfRange() {
	cal, err := timex.NewUmmAlQura()
	if err != nil {
		panic(err)
	}

	_, err = cal.FromCivilDate(time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC), time.UTC)
	fmt.Println(errors.Is(err, timex.ErrHijriOutOfRange))
	// Output: true
}

func ExampleUmmAlQura_MonthLength() {
	cal, err := timex.NewUmmAlQura()
	if err != nil {
		panic(err)
	}

	fmt.Println(cal.MonthLength(1448, 5))
	fmt.Println(cal.MonthLength(1501, 1))
	// Output:
	// 30
	// 0
}

func ExampleParseISODuration() {
	d, err := timex.ParseISODuration("PT1H2M3S") // YouTube contentDetails.duration
	if err != nil {
		panic(err)
	}
	fmt.Println(d)

	_, err = timex.ParseISODuration("P1Y")
	fmt.Println(errors.Is(err, timex.ErrInvalidISODuration))
	// Output:
	// 1h2m3s
	// true
}

func ExampleUmmAlQura_ToCivilDate() {
	// 1 Rajab 1448 H, first by the table and then a day later by sighting.
	d := timex.HijriDate{Year: 1448, Month: 7, Day: 1}
	cal, err := timex.NewUmmAlQura()
	if err != nil {
		panic(err)
	}
	moved, err := timex.NewUmmAlQura(timex.WithHijriMonthOffsets(
		timex.HijriMonthOffset{Year: 1448, Month: 7, Days: 1},
	))
	if err != nil {
		panic(err)
	}

	for _, c := range []*timex.UmmAlQura{cal, moved} {
		t, err := c.ToCivilDate(d, timex.Jakarta)
		if err != nil {
			panic(err)
		}
		fmt.Println(t.Format(time.DateOnly))
	}
	// Output:
	// 2026-12-10
	// 2026-12-11
}

func ExampleHijriMonthLengthError() {
	// Moving 1 Jumadil Akhirah 1448 H a day later makes Jumadil Ula 31 days.
	_, err := timex.NewUmmAlQura(timex.WithHijriMonthOffsets(
		timex.HijriMonthOffset{Year: 1448, Month: 6, Days: 1},
	))
	var lengthErr *timex.HijriMonthLengthError
	if errors.As(err, &lengthErr) {
		fmt.Println(lengthErr.Month, lengthErr.Year, lengthErr.Days)
	}
	// Output: 5 1448 31
}
