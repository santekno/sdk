package validx

import (
	"regexp"
	"unicode"

	"github.com/go-playground/validator/v10"
)

var (
	reIDPhone  = regexp.MustCompile(`^(\+62|62|0)(8[1-9][0-9]{6,10}|2[0-9]{6,11}|[3-9][0-9]{5,10})$`)
	reIDPostal = regexp.MustCompile(`^[1-9][0-9]{4}$`)
)

// registerTags registers all Indonesian validation tags on v.
func registerTags(v *validator.Validate) {
	_ = v.RegisterValidation("nik", validateNIKTag)
	_ = v.RegisterValidation("npwp", validateNPWPTag)
	_ = v.RegisterValidation("id_phone", validateIDPhone)
	_ = v.RegisterValidation("id_postal", validateIDPostal)
}

// validateNIKTag validates a 16-digit NIK structurally (digits only, correct length).
func validateNIKTag(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	if len(s) != 16 {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// validateNPWPTag validates a 15-digit NPWP (strips formatting first).
func validateNPWPTag(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	digits := stripNonDigits(s)
	if len(digits) != 15 {
		return false
	}
	for _, r := range digits {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// validateIDPhone validates Indonesian phone numbers (mobile and landline).
func validateIDPhone(fl validator.FieldLevel) bool {
	return reIDPhone.MatchString(fl.Field().String())
}

// validateIDPostal validates Indonesian postal codes (5-digit, starting 1-9).
func validateIDPostal(fl validator.FieldLevel) bool {
	return reIDPostal.MatchString(fl.Field().String())
}

// stripNonDigits removes all non-digit characters from s.
func stripNonDigits(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if unicode.IsDigit(r) {
			out = append(out, r)
		}
	}
	return string(out)
}
