package gin_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	ginmw "github.com/santekno/sdk/middleware/gin"
	"github.com/santekno/sdk/jwtx"
	"github.com/santekno/sdk/logx"
)

func init() {
	gin.SetMode(gin.TestMode)
}

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

	r := gin.New()
	r.GET("/me", ginmw.JWTAuth(key), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestJWTAuth_MissingHeader(t *testing.T) {
	key := jwtx.HS256([]byte("test-secret"))

	r := gin.New()
	r.GET("/me", ginmw.JWTAuth(key), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	key := jwtx.HS256([]byte("test-secret"))

	r := gin.New()
	r.GET("/me", ginmw.JWTAuth(key), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestLoggerMiddleware(t *testing.T) {
	l := logx.Noop()
	r := gin.New()
	r.Use(ginmw.Logger(l))
	r.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	l := logx.Noop()
	r := gin.New()
	r.Use(ginmw.Recovery(l))
	r.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

func TestTracingMiddleware(t *testing.T) {
	r := gin.New()
	r.Use(ginmw.Tracing())
	r.GET("/trace", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/trace", nil)
	req.Header.Set("X-Request-ID", "my-req-id")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if got := w.Header().Get("X-Request-ID"); got != "my-req-id" {
		t.Errorf("X-Request-ID = %q, want my-req-id", got)
	}
}
