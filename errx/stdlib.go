package errx

import "errors"

// Is reports whether any error in err's tree matches target.
// Delegates to errors.Is.
var Is = errors.Is

// As finds the first error in err's tree that matches target.
// Delegates to errors.As.
var As = errors.As

// Join returns an error that wraps the given errors.
// Delegates to errors.Join (Go 1.20+).
var Join = errors.Join
