package httpx

import "errors"

// Sentinel errors returned by [Client] methods.
// Use [errors.Is] to check for specific conditions.
var (
	// ErrTimeout is returned when the request exceeds the configured timeout.
	ErrTimeout = errors.New("santekno/httpx: timeout")

	// ErrBreakerOpen is returned when the circuit breaker is open and the
	// request was not attempted.
	ErrBreakerOpen = errors.New("santekno/httpx: circuit breaker open")

	// ErrMaxRetries is returned when all retry attempts have been exhausted.
	ErrMaxRetries = errors.New("santekno/httpx: max retries exceeded")

	// ErrNon2xx is returned when a response was received but the HTTP status
	// code is outside the 2xx–3xx range.
	ErrNon2xx = errors.New("santekno/httpx: non-2xx response")
)
