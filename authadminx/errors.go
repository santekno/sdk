package authadminx

import (
	"errors"
	"fmt"
	"io"
	"net/http"
)

// Sentinel errors, matchable with errors.Is on an *APIError.
var (
	ErrUnauthorized  = errors.New("authadminx: unauthorized")
	ErrNotFound      = errors.New("authadminx: not found")
	ErrConflict      = errors.New("authadminx: conflict")
	ErrUnprocessable = errors.New("authadminx: unprocessable")
	ErrRateLimited   = errors.New("authadminx: rate limited")
	// ErrEmailExists matches an *APIError whose Code reports that the email is
	// already registered ("email_exists" or "user_already_exists"), whatever
	// the status. Invite returns it (422 email_exists) for a confirmed user.
	ErrEmailExists = errors.New("authadminx: email exists")

	// ErrResponseTooLarge is returned when a response body exceeds 1 MiB.
	ErrResponseTooLarge = errors.New("authadminx: response too large")
	// ErrInvalidArgument is returned for input rejected before any request is sent.
	ErrInvalidArgument = errors.New("authadminx: invalid argument")
	// ErrInvalidResponse is returned when a success response cannot be decoded.
	ErrInvalidResponse = errors.New("authadminx: invalid response")
	// ErrTransport matches errors raised before a complete response was read:
	// dial, DNS, TLS, timeout, cancellation, or a broken response body. The
	// underlying cause stays matchable (errors.Is with context.Canceled or
	// context.DeadlineExceeded, errors.As with net.Error), but the request URL
	// is dropped and the service role key never appears in the text.
	ErrTransport = errors.New("authadminx: transport error")
)

// APIError is a non-2xx answer from the Auth server. Message is built from the
// status and Code only; the raw response body is never exposed.
type APIError struct {
	Status  int    // HTTP status
	Code    string // GoTrue error_code such as "email_exists"; empty if absent or unsafe
	Message string
}

// Error implements error.
func (e *APIError) Error() string { return e.Message }

// Is maps the status to the sentinel errors.
func (e *APIError) Is(target error) bool {
	switch target {
	case ErrUnauthorized:
		return e.Status == http.StatusUnauthorized || e.Status == http.StatusForbidden
	case ErrNotFound:
		return e.Status == http.StatusNotFound
	case ErrConflict:
		return e.Status == http.StatusConflict
	case ErrUnprocessable:
		return e.Status == http.StatusUnprocessableEntity
	case ErrRateLimited:
		return e.Status == http.StatusTooManyRequests
	case ErrEmailExists:
		return e.Code == "email_exists" || e.Code == "user_already_exists"
	}
	return false
}

func newAPIError(status int, code string) *APIError {
	msg := fmt.Sprintf("authadminx: %d %s", status, http.StatusText(status))
	if code != "" {
		msg += " (" + code + ")"
	}
	return &APIError{Status: status, Code: code, Message: msg}
}

// transportError wraps a failure that happened before a complete response was
// read. It matches ErrTransport and its (already redacted) cause.
type transportError struct {
	stage string
	err   error
}

func (e *transportError) Error() string { return "authadminx: " + e.stage + ": " + e.err.Error() }

func (e *transportError) Unwrap() []error { return []error{ErrTransport, e.err} }

// Format prints the same text for every verb, including %+v and %#v.
func (e *transportError) Format(f fmt.State, _ rune) { _, _ = io.WriteString(f, e.Error()) }

// redactedError replaces a cause whose text contained the service role key.
// It keeps context.Canceled/context.DeadlineExceeded matchable and reports
// Timeout for net.Error checks.
type redactedError struct {
	msg     string
	timeout bool
	cause   error
}

func (e *redactedError) Error() string   { return e.msg }
func (e *redactedError) Unwrap() error   { return e.cause }
func (e *redactedError) Timeout() bool   { return e.timeout }
func (e *redactedError) Temporary() bool { return e.timeout }

func (e *redactedError) Format(f fmt.State, _ rune) { _, _ = io.WriteString(f, e.msg) }
