package s3

import (
	"context"
	"errors"
	"net"
	"regexp"
	"strconv"
)

var (
	// ErrInvalidConfig is matched by New errors. The message names the field,
	// never its value.
	ErrInvalidConfig = errors.New("storagex/s3: invalid config")
	// ErrInvalidRequest is matched by PresignPut errors for a content type or
	// TTL it cannot sign. Invalid keys return storagex.ErrInvalidKey.
	ErrInvalidRequest = errors.New("storagex/s3: invalid request")
	// ErrTransport is matched by errors where no HTTP response was received
	// (connection, TLS, timeout, cancellation). They are usually retryable.
	ErrTransport = errors.New("storagex/s3: transport error")
)

// Error is returned by Client operations that reached the transport.
//
// Its message carries only the operation, the HTTP status, and the S3 error
// code. It never includes the request URL, the object key, headers (such as
// Authorization), credentials, the response message, or the underlying
// transport error text, which can embed the URL.
type Error struct {
	// Op is "put", "get", "delete", "stat", or "list".
	Op string
	// StatusCode is the HTTP status, or 0 when no response was received.
	StatusCode int
	// Code is the S3 error code from the response body (e.g. "AccessDenied"),
	// when present and well-formed.
	Code string

	sentinel error // storagex.ErrNotFound, storagex.ErrAlreadyExists, or nil
	cause    error // context.Canceled or context.DeadlineExceeded, or nil
	timeout  bool
	detail   string // fixed text chosen by this package
}

// Error implements error.
func (e *Error) Error() string {
	msg := "storagex/s3: " + e.Op + ": "
	switch {
	case e.StatusCode == 0:
		msg += "transport error"
		if e.detail != "" {
			msg += ": " + e.detail
		}
	case e.sentinel != nil:
		msg += e.detail + " (status " + strconv.Itoa(e.StatusCode)
		if e.Code != "" {
			msg += " " + e.Code
		}
		msg += ")"
	default:
		msg += "status " + strconv.Itoa(e.StatusCode)
		if e.Code != "" {
			msg += " " + e.Code
		}
		if e.detail != "" {
			msg += ": " + e.detail
		}
	}
	return msg
}

// GoString returns the same text as Error, so %#v prints no internals.
func (e *Error) GoString() string { return e.Error() }

// Unwrap returns storagex.ErrNotFound or storagex.ErrAlreadyExists for those
// outcomes, and context.Canceled or context.DeadlineExceeded when the request
// was canceled.
func (e *Error) Unwrap() error {
	if e.sentinel != nil {
		return e.sentinel
	}
	return e.cause
}

// Is reports ErrTransport for errors without an HTTP response.
func (e *Error) Is(target error) bool { return target == ErrTransport && e.StatusCode == 0 }

// Timeout reports whether the transport timed out (net.Error compatible).
func (e *Error) Timeout() bool { return e.timeout }

// transportError classifies err from http.Client.Do without keeping its text.
func transportError(op string, err error) *Error {
	e := &Error{Op: op}
	var ne net.Error
	switch {
	case errors.Is(err, context.Canceled):
		e.cause, e.detail = context.Canceled, "canceled"
	case errors.Is(err, context.DeadlineExceeded):
		e.cause, e.detail, e.timeout = context.DeadlineExceeded, "deadline exceeded", true
	case errors.As(err, &ne) && ne.Timeout():
		e.detail, e.timeout = "timeout", true
	}
	return e
}

var errorCodePattern = regexp.MustCompile(`^[A-Za-z0-9.]{1,64}$`)

// safeCode returns code when it looks like an S3 error code, else "".
func safeCode(code string) string {
	if errorCodePattern.MatchString(code) {
		return code
	}
	return ""
}
