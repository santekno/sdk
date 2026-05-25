// Package testfixture provides shared test data for SDK packages.
// Import only in _test.go files.
package testfixture

// ValidNIKs contains synthetic NIK strings that pass structural validation.
var ValidNIKs = []string{
	"3201010101800001",
	"3271010101750002",
	"3374020202990003",
	"1101030303850004",
	"5101040404960005",
}

// InvalidNIKs contains NIK strings that must fail ParseNIK.
var InvalidNIKs = []string{
	"",
	"123",
	"320101010180000X",
	"0000000000000000", // province 00 not in registry
	"9999999999999999", // province 99 not in registry
	"3201010001800001", // month 00 invalid
	"3201013201800001", // day > 72 invalid for female too
}

// ValidNPWPs contains synthetic NPWP strings that pass structural validation.
var ValidNPWPs = []string{
	"012345678901000",
	"987654321098765",
}

// InvalidNPWPs contains NPWP strings that must fail ParseNPWP.
var InvalidNPWPs = []string{
	"",
	"01234",
	"01234567890100X",
}
