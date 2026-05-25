package errx

import "errors"

// Code returns the Code field of the first AppError found in the error chain.
// Returns the empty string if no AppError is found.
func Code(err error) string {
	var e *AppError
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

// Details returns the Details map of the first AppError found in the error chain.
// Returns nil if no AppError is found.
func Details(err error) map[string]any {
	var e *AppError
	if errors.As(err, &e) {
		return e.Details
	}
	return nil
}

// HTTPStatus returns the HTTPStatus of the first AppError found in the error chain.
// Returns 0 if no AppError is found or if HTTPStatus was not set.
func HTTPStatus(err error) int {
	var e *AppError
	if errors.As(err, &e) {
		return e.HTTPStatus
	}
	return 0
}
