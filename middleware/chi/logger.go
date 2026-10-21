package chi

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/santekno/sdk/ctxx"
	"github.com/santekno/sdk/logx"
)

// Logger returns middleware that writes one structured line per request after
// it completes: method, path, chi route pattern, status, bytes, duration and
// request ID. Put it after RequestID so the ID is available.
func Logger(l logx.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			status := ww.Status()
			if status == 0 {
				status = http.StatusOK
			}
			l.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"route", routePattern(r),
				"status", status,
				"bytes", ww.BytesWritten(),
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", ctxx.RequestID(r.Context()),
				"remote_addr", r.RemoteAddr,
			)
		})
	}
}

// routePattern is the matched chi pattern ("/products/{id}"), which keeps log
// and span cardinality low; it is empty when no chi route matched.
func routePattern(r *http.Request) string {
	if rc := chi.RouteContext(r.Context()); rc != nil {
		return rc.RoutePattern()
	}
	return ""
}
