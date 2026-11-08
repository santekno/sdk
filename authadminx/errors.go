package authadminx

import (
	"errors"
	"fmt"
	"net/http"
)

// Sentinel errors, matchable with errors.Is on an *APIError.
var (
	ErrUnauthorized  = errors.New("authadminx: unauthorized")
	ErrNotFound      = errors.New("authadminx: not found")
	ErrConflict      = errors.New("authadminx: conflict")
	ErrUnprocessable = errors.New("authadminx: unprocessable")
	ErrRateLimited   = errors.New("authadminx: rate limited")

	// ErrResponseTooLarge is returned when a response body exceeds 1 MiB.
	ErrResponseTooLarge = errors.New("authadminx: response too large")
	// ErrInvalidArgument is returned for input rejected before any request is sent.
	ErrInvalidArgument = errors.New("authadminx: invalid argument")
	// ErrInvalidResponse is returned when a success response cannot be decoded.
	ErrInvalidResponse = errors.New("authadminx: invalid response")
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
