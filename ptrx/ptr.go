// Package ptrx provides generic pointer helper utilities.
//
//	p := ptrx.Of(42)          // *int pointing to 42
//	v := ptrx.Deref(p, 0)    // 42
//	v  = ptrx.Deref(nil, 0)  // 0 (default)
package ptrx

// Of returns a pointer to a copy of v.
func Of[T any](v T) *T { return &v }

// Deref returns *p if p is non-nil, otherwise it returns def.
func Deref[T any](p *T, def T) T {
	if p == nil {
		return def
	}
	return *p
}

// IsNil reports whether p is a nil pointer.
func IsNil[T any](p *T) bool { return p == nil }

// Equal reports whether two pointers point to equal values.
// Two nil pointers are equal. A nil and non-nil pointer are not equal.
func Equal[T comparable](a, b *T) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
