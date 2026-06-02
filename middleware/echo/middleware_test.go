package echo_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/santekno/sdk/ctxx"
	"github.com/santekno/sdk/jwtx"
	"github.com/santekno/sdk/logx"
	echomw "github.com/santekno/sdk/middleware/echo"
)

// silentLogger is logx.Noop() — discards everything. Both tests use this so
// no log noise pollutes test output; we don't need to inspect log fields
// since the Echo middleware just delegates.

// ============================================================================
// JWTAuth
// ============================================================================

func TestJWTAuth_ValidToken(t *testing.T) {
	key := jwtx.HS256([]byte("test-secret"))
	claims := jwtx.Claims{
		Subject:   "user-1",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	token, err := jwtx.Sign(claims, key)
	if err != nil {
		t.Fatal(err)
	}

	e := echo.New()
	e.GET("/me", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	}, echomw.JWTAuth(key))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	e.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestJWTAuth_MissingHeader(t *testing.T) {
	key := jwtx.HS256([]byte("test-secret"))

	e := echo.New()
	e.GET("/me", func(c echo.Context) error { return c.NoContent(http.StatusOK) }, echomw.JWTAuth(key))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	e.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestJWTAuth_InvalidFormat(t *testing.T) {
	key := jwtx.HS256([]byte("test-secret"))

	e := echo.New()
	e.GET("/me", func(c echo.Context) error { return c.NoContent(http.StatusOK) }, echomw.JWTAuth(key))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "NotBearer xxx")
	e.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	key := jwtx.HS256([]byte("test-secret"))

	e := echo.New()
	e.GET("/me", func(c echo.Context) error { return c.NoContent(http.StatusOK) }, echomw.JWTAuth(key))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer not-a-valid-token")
	e.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestJWTAuth_PropagatesUserID(t *testing.T) {
	key := jwtx.HS256([]byte("test-secret"))
	claims := jwtx.Claims{
		Subject:   "user-42",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	token, _ := jwtx.Sign(claims, key)

	var seenUserID string
	e := echo.New()
	e.GET("/me", func(c echo.Context) error {
		seenUserID = ctxx.UserID(c.Request().Context())
		return c.NoContent(http.StatusOK)
	}, echomw.JWTAuth(key))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	e.ServeHTTP(w, req)

	if seenUserID != "user-42" {
		t.Errorf("ctxx.UserID = %q, want %q", seenUserID, "user-42")
	}
}

// ============================================================================
// Logger
// ============================================================================

func TestLoggerMiddleware(t *testing.T) {
	log := logx.Noop()

	e := echo.New()
	e.GET("/ping", func(c echo.Context) error {
		return c.String(http.StatusOK, "pong")
	}, echomw.Logger(log))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	e.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if w.Body.String() != "pong" {
		t.Errorf("body = %q, want %q", w.Body.String(), "pong")
	}
	// Logger middleware delegates to logx.Logger.Info — the noop logger
	// discards fields. The test asserts only that the request flowed through
	// the middleware without altering the handler's output.
}

// ============================================================================
// Recovery
// ============================================================================

func TestRecoveryMiddleware(t *testing.T) {
	log := logx.Noop()

	e := echo.New()
	e.GET("/boom", func(c echo.Context) error {
		panic("synthetic panic")
	}, echomw.Recovery(log))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	e.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
	if !strings.Contains(w.Body.String(), "internal server error") {
		t.Errorf("body = %q, want it to contain 'internal server error'", w.Body.String())
	}
}

// ============================================================================
// Tracing
// ============================================================================

func TestTracingMiddleware_GeneratesIDWhenMissing(t *testing.T) {
	var seenID string
	e := echo.New()
	e.GET("/id", func(c echo.Context) error {
		seenID = ctxx.RequestID(c.Request().Context())
		return c.NoContent(http.StatusOK)
	}, echomw.Tracing())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/id", nil)
	e.ServeHTTP(w, req)

	if seenID == "" {
		t.Error("expected generated request ID")
	}
	if w.Header().Get("X-Request-ID") != seenID {
		t.Errorf("response X-Request-ID = %q, want %q", w.Header().Get("X-Request-ID"), seenID)
	}
}

func TestTracingMiddleware_PropagatesIncomingID(t *testing.T) {
	var seenID string
	e := echo.New()
	e.GET("/id", func(c echo.Context) error {
		seenID = ctxx.RequestID(c.Request().Context())
		return c.NoContent(http.StatusOK)
	}, echomw.Tracing())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/id", nil)
	req.Header.Set("X-Request-ID", "incoming-abc-123")
	e.ServeHTTP(w, req)

	if seenID != "incoming-abc-123" {
		t.Errorf("expected incoming request ID, got %q", seenID)
	}
	if w.Header().Get("X-Request-ID") != "incoming-abc-123" {
		t.Errorf("response X-Request-ID echo failed")
	}
}

func TestTracingMiddleware_FallsBackToTraceparent(t *testing.T) {
	var seenID string
	e := echo.New()
	e.GET("/id", func(c echo.Context) error {
		seenID = ctxx.RequestID(c.Request().Context())
		return c.NoContent(http.StatusOK)
	}, echomw.Tracing())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/id", nil)
	req.Header.Set("traceparent", "00-trace-id-span-id-01")
	e.ServeHTTP(w, req)

	if seenID != "00-trace-id-span-id-01" {
		t.Errorf("expected traceparent fallback, got %q", seenID)
	}
}
