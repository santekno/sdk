package chi_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/santekno/sdk/ctxx"
	"github.com/santekno/sdk/jwtx"
	"github.com/santekno/sdk/logx"
	chimw "github.com/santekno/sdk/middleware/chi"
	"github.com/santekno/sdk/tracex"
)

// A chi router with the full middleware chain and one protected route.
func Example() {
	l := logx.Noop()
	key := jwtx.HS256([]byte("change-me"))

	r := chi.NewRouter()
	r.Use(chimw.RequestID(), chimw.Logger(l), chimw.Recovery(l), chimw.Tracing(tracex.Global()))
	r.With(chimw.JWTAuth(key)).Get("/me", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, "hello %s", ctxx.UserID(r.Context()))
	})

	token, _ := jwtx.Sign(jwtx.Claims{Subject: "user-42", ExpiresAt: time.Now().Add(time.Hour)}, key)

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	fmt.Println(w.Code, w.Body.String())

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/me", nil))
	fmt.Print(w.Code, " ", w.Body.String())
	// Output:
	// 200 hello user-42
	// 401 {"error":{"code":"UNAUTHORIZED","message":"missing Authorization header"}}
}
