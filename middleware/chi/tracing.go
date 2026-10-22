package chi

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/santekno/sdk/tracex"
)

// Tracing returns middleware that starts one span per request with t (use
// tracex.Global() for the process-wide tracer). The span is renamed to
// "METHOD /route/{pattern}" once chi has matched the route, and records the
// method, route, status and request ID. 5xx responses mark the span as error.
func Tracing(t tracex.Tracer) func(http.Handler) http.Handler {
	if t == nil {
		t = tracex.NoopTracer()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := t.Start(r.Context(), "http "+r.Method)
			defer span.End()

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r.WithContext(ctx))

			status := ww.Status()
			if status == 0 {
				status = http.StatusOK
			}
			route := routePattern(r)
			span.SetAttribute("http.method", r.Method)
			span.SetAttribute("http.route", route)
			span.SetAttribute("http.status_code", status)
			span.SetAttribute("http.request_id", w.Header().Get(HeaderRequestID))
			if status >= http.StatusInternalServerError {
				span.SetStatus("error", http.StatusText(status))
			} else {
				span.SetStatus("ok", "")
			}
		})
	}
}
