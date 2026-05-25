package gin

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/santekno/sdk/ctxx"
	"github.com/santekno/sdk/jwtx"
)

// JWTAuth returns a Gin middleware that validates Bearer tokens.
//
// On success the validated claims are stored on the request context via ctxx.WithUserID.
// On failure the request is aborted with HTTP 401.
func JWTAuth(key jwtx.Key) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing Authorization header"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid Authorization header format"})
			return
		}

		claims, err := jwtx.Verify(parts[1], key)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		ctx := ctxx.WithUserID(c.Request.Context(), claims.Subject)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
