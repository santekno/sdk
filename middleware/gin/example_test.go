package gin_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gin-gonic/gin"
	ginmw "github.com/santekno/sdk/middleware/gin"
	"github.com/santekno/sdk/jwtx"
	"github.com/santekno/sdk/logx"
)

func ExampleJWTAuth() {
	gin.SetMode(gin.TestMode)

	key := jwtx.HS256([]byte("example-secret"))
	token, _ := jwtx.Sign(jwtx.Claims{
		Subject:   "user-1",
		ExpiresAt: time.Now().Add(time.Hour),
	}, key)

	r := gin.New()
	r.GET("/secure", ginmw.JWTAuth(key), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	fmt.Println(w.Code)
	// Output: 200
}

func ExampleLogger() {
	gin.SetMode(gin.TestMode)

	l := logx.Noop()
	r := gin.New()
	r.Use(ginmw.Logger(l))
	r.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ping", nil))
	fmt.Println(w.Code)
	// Output: 200
}

func ExampleRecovery() {
	gin.SetMode(gin.TestMode)

	l := logx.Noop()
	r := gin.New()
	r.Use(ginmw.Recovery(l))
	r.GET("/crash", func(c *gin.Context) { panic("boom") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/crash", nil))
	fmt.Println(w.Code)
	// Output: 500
}
