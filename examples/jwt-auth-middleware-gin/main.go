// Example: jwt-auth-middleware-gin demonstrates a Gin server with JWT middleware,
// logx structured logging, and hashx password validation.
package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/santekno/sdk/ctxx"
	"github.com/santekno/sdk/hashx"
	"github.com/santekno/sdk/jwtx"
	"github.com/santekno/sdk/logx"
	ginmw "github.com/santekno/sdk/middleware/gin"
)

var signingKey = jwtx.HS256([]byte("super-secret-key-change-in-production"))

func main() {
	logger := logx.New(logx.WithFormat("console"), logx.WithLevel("debug"))

	r := gin.New()
	r.Use(ginmw.Logger(logger), ginmw.Recovery(logger), ginmw.Tracing())

	r.POST("/login", loginHandler)
	r.GET("/profile", ginmw.JWTAuth(signingKey), profileHandler)

	logger.Info("server starting", "addr", ":8080")
	if err := r.Run(":8080"); err != nil {
		logger.Error("server failed", "error", err)
	}
}

func loginHandler(c *gin.Context) {
	var req struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// In production: load hash from database.
	storedHash, _ := hashx.HashPassword("demo-password")
	if !hashx.VerifyPassword(req.Password, storedHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid password"})
		return
	}

	claims := jwtx.Claims{Subject: "demo-user"}
	token, err := jwtx.Sign(claims, signingKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token signing failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

func profileHandler(c *gin.Context) {
	userID := ctxx.UserID(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		"message": fmt.Sprintf("Hello, %s!", userID),
	})
}
