package chi

import (
	"errors"
	"net/http"
	"runtime/debug"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/santekno/sdk/ctxx"
	"github.com/santekno/sdk/errx"
	"github.com/santekno/sdk/logx"
)

// Recovery returns middleware that recovers from a panic in later handlers,
// logs the panic value with its stack, and answers 500 INTERNAL_ERROR, unless
// the handler already started writing the response. http.ErrAbortHandler is
// re-panicked so net/http can abort the connection as intended.
func Recovery(l logx.Logger, opts ...Option) func(http.Handler) http.Handler {
	o := newOptions(opts)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			defer recoverPanic(ww, r, l, o)
			next.ServeHTTP(ww, r)
		})
	}
}

// recoverPanic must be deferred directly so recover() sees the panic.
func recoverPanic(ww middleware.WrapResponseWriter, r *http.Request, l logx.Logger, o *options) {
	rec := recover()
	if rec == nil {
		return
	}
	if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
		panic(rec)
	}
	l.Error("panic recovered",
		"error", rec,
		"method", r.Method,
		"path", r.URL.Path,
		"request_id", ctxx.RequestID(r.Context()),
		"stack", string(debug.Stack()),
	)
	if ww.Status() == 0 {
		o.writeError(ww, r, errx.New("INTERNAL_ERROR", "internal server error").
			WithHTTPStatus(http.StatusInternalServerError))
	}
}
