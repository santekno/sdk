package errx

import (
	"fmt"
	"runtime"
)

// Frame represents a single stack frame captured when an AppError is created.
type Frame struct {
	File     string
	Line     int
	Function string
}

// AppError is the structured error type used across the Santekno Go SDK.
// It carries a machine-readable Code, a human-readable Message,
// an optional wrapped Cause, structured Details, an HTTP status, and a stack trace.
type AppError struct {
	// Code is a SCREAMING_SNAKE_CASE machine-readable identifier, e.g. "NIK_INVALID_LENGTH".
	Code string
	// Message is a human-readable, safe-to-log description.
	Message string
	// Cause is the wrapped underlying error (accessible via errors.Unwrap).
	Cause error
	// Details holds structured extra context (field names, counts, etc.).
	Details map[string]any
	// Stack contains the call frames captured at the error creation site.
	Stack []Frame
	// HTTPStatus is an optional HTTP status code mapping (0 = not set).
	HTTPStatus int
}

// Error implements the error interface. Returns "CODE: Message".
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the wrapped cause so errors.Is / errors.As traverse the chain.
func (e *AppError) Unwrap() error { return e.Cause }

// New creates an AppError with the given code and message.
// A stack trace is captured at the call site.
func New(code, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Stack:   captureStack(1),
	}
}

// Newf creates an AppError with a formatted message.
func Newf(code, format string, args ...any) *AppError {
	return &AppError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Stack:   captureStack(1),
	}
}

// Wrap wraps an existing error as the Cause of a new AppError.
func Wrap(err error, code, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Cause:   err,
		Stack:   captureStack(1),
	}
}

// Wrapf wraps an existing error with a formatted message.
func Wrapf(err error, code, format string, args ...any) *AppError {
	return &AppError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Cause:   err,
		Stack:   captureStack(1),
	}
}

func captureStack(skip int) []Frame {
	pcs := make([]uintptr, 32)
	n := runtime.Callers(skip+2, pcs)
	frames := runtime.CallersFrames(pcs[:n])
	result := make([]Frame, 0, n)
	for {
		f, more := frames.Next()
		result = append(result, Frame{
			File:     f.File,
			Line:     f.Line,
			Function: f.Function,
		})
		if !more {
			break
		}
	}
	return result
}
