package chi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/santekno/sdk/ctxx"
	"github.com/santekno/sdk/errx"
	"github.com/santekno/sdk/jwtx"
	"github.com/santekno/sdk/logx"
	chimw "github.com/santekno/sdk/middleware/chi"
	"github.com/santekno/sdk/tracex"
)

var testKey = jwtx.HS256([]byte("test-secret"))

func signToken(t *testing.T, sub string, exp time.Time) string {
	t.Helper()
	tok, err := jwtx.Sign(jwtx.Claims{Subject: sub, ExpiresAt: exp}, testKey)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func serve(h http.Handler, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func decodeError(t *testing.T, w *httptest.ResponseRecorder) (code, message string) {
	t.Helper()
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	return body.Error.Code, body.Error.Message
}

func TestJWTAuth(t *testing.T) {
	valid := signToken(t, "user-1", time.Now().Add(time.Hour))
	expired := signToken(t, "user-1", time.Now().Add(-time.Hour))

	cases := []struct {
		name       string
		header     string
		wantStatus int
		wantMsg    string
	}{
		{"valid token", "Bearer " + valid, http.StatusOK, ""},
		{"lowercase scheme", "bearer " + valid, http.StatusOK, ""},
		{"missing header", "", http.StatusUnauthorized, "missing Authorization header"},
		{"wrong scheme", "Basic " + valid, http.StatusUnauthorized, "invalid Authorization header format"},
		{"no token", "Bearer ", http.StatusUnauthorized, "invalid Authorization header format"},
		{"no space", "Bearer" + valid, http.StatusUnauthorized, "invalid Authorization header format"},
		{"expired", "Bearer " + expired, http.StatusUnauthorized, "invalid or expired token"},
		{"tampered", "Bearer " + valid + "x", http.StatusUnauthorized, "invalid or expired token"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var gotUser, gotSub string
			h := chimw.JWTAuth(testKey)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotUser = ctxx.UserID(r.Context())
				if cl, ok := chimw.ClaimsFromContext(r.Context()); ok {
					gotSub = cl.Subject
				}
				w.WriteHeader(http.StatusOK)
			}))
			req := httptest.NewRequest(http.MethodGet, "/me", nil)
			if c.header != "" {
				req.Header.Set("Authorization", c.header)
			}
			w := serve(h, req)

			if w.Code != c.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, c.wantStatus)
			}
			if c.wantStatus == http.StatusOK {
				if gotUser != "user-1" || gotSub != "user-1" {
					t.Fatalf("user id %q / claims subject %q, want user-1", gotUser, gotSub)
				}
				return
			}
			code, msg := decodeError(t, w)
			if code != "UNAUTHORIZED" || msg != c.wantMsg {
				t.Fatalf("error = %s %q, want UNAUTHORIZED %q", code, msg, c.wantMsg)
			}
			if gotUser != "" {
				t.Fatal("next handler ran for an unauthorized request")
			}
		})
	}
}

func TestClaimsFromContextEmpty(t *testing.T) {
	if _, ok := chimw.ClaimsFromContext(context.Background()); ok {
		t.Fatal("ClaimsFromContext on an empty context returned ok")
	}
}

func TestWithErrorWriter(t *testing.T) {
	var got *errx.AppError
	custom := func(w http.ResponseWriter, _ *http.Request, err *errx.AppError) {
		got = err
		w.WriteHeader(http.StatusTeapot)
	}
	h := chimw.JWTAuth(testKey, chimw.WithErrorWriter(custom))(http.NotFoundHandler())
	w := serve(h, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusTeapot || got == nil || got.Code != "UNAUTHORIZED" {
		t.Fatalf("custom writer not used: status %d, err %v", w.Code, got)
	}

	// A nil writer keeps the default instead of panicking.
	h = chimw.JWTAuth(testKey, chimw.WithErrorWriter(nil))(http.NotFoundHandler())
	if w := serve(h, httptest.NewRequest(http.MethodGet, "/", nil)); w.Code != http.StatusUnauthorized {
		t.Fatalf("nil writer: status = %d, want 401", w.Code)
	}
}

func TestDefaultErrorWriterWithoutStatus(t *testing.T) {
	w := httptest.NewRecorder()
	chimw.DefaultErrorWriter(w, httptest.NewRequest(http.MethodGet, "/", nil), errx.New("BOOM", "boom"))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("content type = %q", ct)
	}
	if code, msg := decodeError(t, w); code != "BOOM" || msg != "boom" {
		t.Fatalf("body = %s %q", code, msg)
	}
}

func TestRequestID(t *testing.T) {
	cases := []struct {
		name, incoming string
		keep           bool
	}{
		{"propagates a valid id", "req-123.abc", true},
		{"generates when missing", "", false},
		{"replaces an unsafe id", "bad id\nwith newline", false},
		{"replaces an oversized id", strings.Repeat("a", 129), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var inCtx string
			h := chimw.RequestID()(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				inCtx = ctxx.RequestID(r.Context())
			}))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if c.incoming != "" {
				req.Header.Set(chimw.HeaderRequestID, c.incoming)
			}
			w := serve(h, req)
			got := w.Header().Get(chimw.HeaderRequestID)

			if got == "" || got != inCtx {
				t.Fatalf("header %q, context %q: want equal and non-empty", got, inCtx)
			}
			if c.keep && got != c.incoming {
				t.Fatalf("id = %q, want propagated %q", got, c.incoming)
			}
			if !c.keep && (got == c.incoming || len(got) != 32) {
				t.Fatalf("id = %q, want a fresh 32-char hex id", got)
			}
		})
	}
}

func newRouter(l logx.Logger, tr tracex.Tracer) *chi.Mux {
	r := chi.NewRouter()
	r.Use(chimw.RequestID(), chimw.Logger(l), chimw.Recovery(l), chimw.Tracing(tr))
	r.Get("/products/{id}", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/boom", func(http.ResponseWriter, *http.Request) { panic("kaboom") })
	r.Get("/late-boom", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		panic("after write")
	})
	r.Get("/fail", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadGateway) })
	r.Get("/empty", func(http.ResponseWriter, *http.Request) {})
	return r
}

func logLines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("log line is not JSON: %q", line)
		}
		out = append(out, m)
	}
	return out
}

func TestLogger(t *testing.T) {
	var buf bytes.Buffer
	r := newRouter(logx.New(logx.WithOutput(&buf), logx.WithFormat("json")), nil)

	req := httptest.NewRequest(http.MethodGet, "/products/42", nil)
	req.Header.Set(chimw.HeaderRequestID, "rid-1")
	if w := serve(r, req); w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	lines := logLines(t, &buf)
	if len(lines) != 1 {
		t.Fatalf("want 1 log line, got %d: %s", len(lines), buf.String())
	}
	l := lines[0]
	want := map[string]any{
		"msg": "http request", "method": "GET", "path": "/products/42",
		"route": "/products/{id}", "status": float64(200), "bytes": float64(2), "request_id": "rid-1",
	}
	for k, v := range want {
		if l[k] != v {
			t.Errorf("log %s = %v, want %v", k, l[k], v)
		}
	}
	if _, ok := l["duration_ms"]; !ok {
		t.Error("log has no duration_ms")
	}
}

func TestLoggerImplicitStatus(t *testing.T) {
	var buf bytes.Buffer
	r := newRouter(logx.New(logx.WithOutput(&buf), logx.WithFormat("json")), nil)
	serve(r, httptest.NewRequest(http.MethodGet, "/empty", nil))
	if got := logLines(t, &buf)[0]["status"]; got != float64(200) {
		t.Fatalf("status logged = %v, want 200 for a handler that writes nothing", got)
	}
}

func TestRecovery(t *testing.T) {
	var buf bytes.Buffer
	r := newRouter(logx.New(logx.WithOutput(&buf), logx.WithFormat("json")), nil)

	w := serve(r, httptest.NewRequest(http.MethodGet, "/boom", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	if code, _ := decodeError(t, w); code != "INTERNAL_ERROR" {
		t.Fatalf("error code = %s", code)
	}

	var panicLine map[string]any
	for _, l := range logLines(t, &buf) {
		if l["msg"] == "panic recovered" {
			panicLine = l
		}
	}
	if panicLine == nil || panicLine["error"] != "kaboom" || !strings.Contains(panicLine["stack"].(string), "goroutine") {
		t.Fatalf("panic not logged with value and stack: %v", panicLine)
	}
}

func TestRecoveryAfterHeadersWritten(t *testing.T) {
	r := newRouter(logx.Noop(), nil)
	w := serve(r, httptest.NewRequest(http.MethodGet, "/late-boom", nil))
	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want the already-written 202", w.Code)
	}
	if strings.Contains(w.Body.String(), "INTERNAL_ERROR") {
		t.Fatal("error body appended after headers were written")
	}
}

func TestRecoveryRepanicsAbortHandler(t *testing.T) {
	h := chimw.Recovery(logx.Noop())(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler)
	}))
	defer func() {
		rec := recover()
		if err, ok := rec.(error); !ok || !errors.Is(err, http.ErrAbortHandler) {
			t.Fatalf("recovered %v, want http.ErrAbortHandler re-panicked", rec)
		}
	}()
	serve(h, httptest.NewRequest(http.MethodGet, "/", nil))
	t.Fatal("ErrAbortHandler was swallowed")
}

type fakeSpan struct {
	attrs          map[string]any
	status, detail string
	ended          bool
}

func (s *fakeSpan) End()                         { s.ended = true }
func (s *fakeSpan) SetAttribute(k string, v any) { s.attrs[k] = v }
func (s *fakeSpan) SetStatus(c, d string)        { s.status, s.detail = c, d }
func (s *fakeSpan) RecordError(error)            {}

type fakeTracer struct{ spans []*fakeSpan }

func (f *fakeTracer) Start(ctx context.Context, _ string, _ ...tracex.SpanStartOption) (context.Context, tracex.Span) {
	s := &fakeSpan{attrs: map[string]any{}}
	f.spans = append(f.spans, s)
	return ctx, s
}

func TestTracing(t *testing.T) {
	cases := []struct {
		path, route string
		status      int
		spanStatus  string
	}{
		{"/products/7", "/products/{id}", http.StatusOK, "ok"},
		{"/fail", "/fail", http.StatusBadGateway, "error"},
		{"/empty", "/empty", http.StatusOK, "ok"},
	}
	for _, c := range cases {
		t.Run(c.path, func(t *testing.T) {
			tr := &fakeTracer{}
			req := httptest.NewRequest(http.MethodGet, c.path, nil)
			req.Header.Set(chimw.HeaderRequestID, "rid-9")
			serve(newRouter(logx.Noop(), tr), req)

			if len(tr.spans) != 1 {
				t.Fatalf("want 1 span, got %d", len(tr.spans))
			}
			s := tr.spans[0]
			if !s.ended || s.status != c.spanStatus {
				t.Fatalf("ended %v status %q, want ended and %q", s.ended, s.status, c.spanStatus)
			}
			if s.attrs["http.route"] != c.route || s.attrs["http.status_code"] != c.status ||
				s.attrs["http.method"] != "GET" || s.attrs["http.request_id"] != "rid-9" {
				t.Fatalf("attrs = %v", s.attrs)
			}
		})
	}
}

func TestTracingNilTracer(t *testing.T) {
	h := chimw.Tracing(nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	if w := serve(h, httptest.NewRequest(http.MethodGet, "/", nil)); w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}
}

func TestDefaultErrorWriterDetails(t *testing.T) {
	w := httptest.NewRecorder()
	err := errx.New("VALIDATION_FAILED", "invalid request").
		WithHTTPStatus(http.StatusUnprocessableEntity).
		WithDetail("field", "email")
	chimw.DefaultErrorWriter(w, httptest.NewRequest(http.MethodPost, "/", nil), err)

	var body struct {
		Error struct {
			Code    string         `json:"code"`
			Details map[string]any `json:"details"`
		} `json:"error"`
	}
	if e := json.NewDecoder(w.Body).Decode(&body); e != nil {
		t.Fatal(e)
	}
	if w.Code != http.StatusUnprocessableEntity || body.Error.Code != "VALIDATION_FAILED" || body.Error.Details["field"] != "email" {
		t.Fatalf("status %d body %+v", w.Code, body)
	}

	// No details: the key is omitted entirely.
	w = httptest.NewRecorder()
	chimw.DefaultErrorWriter(w, httptest.NewRequest(http.MethodGet, "/", nil), errx.New("X", "y"))
	if strings.Contains(w.Body.String(), "details") {
		t.Fatalf("details present without any: %s", w.Body.String())
	}
}
