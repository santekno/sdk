package validx_test

import (
	"fmt"

	"github.com/santekno/sdk/validx"
)

func ExampleParseNIK() {
	info, err := validx.ParseNIK("3201010101800001")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(info.Province)
	// Output: Jawa Barat
}

func ExampleFormatIDR() {
	fmt.Println(validx.FormatIDR(1_500_000))
	// Output: Rp1.500.000
}

func ExampleValidator_Struct() {
	type Form struct {
		NIK  string `validate:"required,nik"`
		NPWP string `validate:"omitempty,npwp"`
	}

	v := validx.New()
	form := Form{NIK: "3201010101800001"}
	err := v.Struct(form)
	fmt.Println(err == nil)
	// Output: true
}
