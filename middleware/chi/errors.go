package chi

import (
	"encoding/json"
	"net/http"

	"github.com/santekno/sdk/errx"
)

// ErrorWriter writes err as the HTTP response. Middleware in this package call
// it whenever they end a request early (401 from JWTAuth, 500 from Recovery).
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err *errx.AppError)

// Option configures a middleware in this package.
type Option func(*options)

type options struct {
	writeError ErrorWriter
}

func newOptions(opts []Option) *options {
	o := &options{writeError: DefaultErrorWriter}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// WithErrorWriter replaces DefaultErrorWriter, so middleware errors use the
// same JSON shape as the rest of the application's API.
func WithErrorWriter(fn ErrorWriter) Option {
	return func(o *options) {
		if fn != nil {
			o.writeError = fn
		}
	}
}

type errorBody struct {
	Error errorPayload `json:"error"`
}

type errorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// DefaultErrorWriter writes {"error":{"code":…,"message":…}} with the error's
// HTTP status (500 when the error carries none).
func DefaultErrorWriter(w http.ResponseWriter, _ *http.Request, err *errx.AppError) {
	status := err.HTTPStatus
	if status == 0 {
		status = http.StatusInternalServerError
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorBody{Error: errorPayload{Code: err.Code, Message: err.Message}})
}
