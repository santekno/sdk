// Example: indonesian-locale demonstrates timex and validx Indonesian locale features.
package main

import (
	"fmt"
	"time"

	"github.com/santekno/sdk/timex"
	"github.com/santekno/sdk/validx"
)

func main() {
	// 1. Format a date in Bahasa Indonesia.
	t := time.Date(2026, time.May, 17, 14, 30, 0, 0, timex.Jakarta)
	fmt.Println("Long date:", timex.FormatID(t, timex.LongID))
	fmt.Println("Short date:", timex.FormatID(t, timex.DateID))

	// 2. Business days between two dates.
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, timex.Jakarta)
	to := time.Date(2026, 1, 31, 0, 0, 0, 0, timex.Jakarta)
	days := timex.BusinessDaysBetween(from, to)
	fmt.Printf("Business days Jan 2026: %d\n", days)

	// 3. Parse a NIK (National ID).
	nik, err := validx.ParseNIK("3201010101800001")
	if err != nil {
		fmt.Println("NIK error:", err)
	} else {
		fmt.Printf("Province: %s, Gender: %s, Birth: %s\n",
			nik.Province, nik.Gender, nik.BirthDate.Format("2006-01-02"))
	}

	// 4. Format Indonesian Rupiah.
	fmt.Println("IDR:", validx.FormatIDR(1_500_000))
	fmt.Println("IDR with cents:", validx.FormatIDR(1_500_000, validx.WithDecimal(true)))

	// 5. Struct validation with Indonesian tags.
	type CitizenForm struct {
		NIK   string `validate:"required,nik"`
		Phone string `validate:"required,id_phone"`
	}
	v := validx.New()
	form := CitizenForm{NIK: "3201010101800001", Phone: "081234567890"}
	if err := v.Struct(form); err != nil {
		fmt.Println("Validation error:", err)
	} else {
		fmt.Println("CitizenForm validation: OK")
	}
}
