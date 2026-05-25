// Package errx provides structured error types for the Santekno Go SDK.
//
// The central type is [AppError], which carries a machine-readable Code,
// a human-readable Message, an optional wrapped Cause, structured Details,
// an HTTP status mapping, and a stack trace.
//
// All SDK packages use AppError as their canonical error type.
// Users can check error codes with [Code] and inspect details with [Details].
//
//	err := errx.New("USER_NOT_FOUND", "user with given ID does not exist").
//	    WithHTTPStatus(http.StatusNotFound).
//	    WithDetail("user_id", id)
//
//	if errx.Code(err) == "USER_NOT_FOUND" {
//	    // handle not-found
//	}
package errx
