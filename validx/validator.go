package validx

import "github.com/go-playground/validator/v10"

// Validator wraps go-playground/validator with Indonesian-specific tags registered.
// Safe for concurrent use.
type Validator struct {
	v *validator.Validate
}

// New returns a *Validator with all Indonesian validation tags registered.
func New() *Validator {
	v := validator.New()
	registerTags(v)
	return &Validator{v: v}
}

// Struct validates all fields of s that carry validate tags.
// Returns nil if all fields pass, or a FieldError slice otherwise.
func (val *Validator) Struct(s any) error {
	return val.v.Struct(s)
}

// Var validates a single field value against the given tag string.
func (val *Validator) Var(field any, tag string) error {
	return val.v.Var(field, tag)
}
