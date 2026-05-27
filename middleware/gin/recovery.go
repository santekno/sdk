package gin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/santekno/sdk/logx"
)

// Recovery returns a Gin middleware that recovers from panics and responds
// with HTTP 500. The panic value and stack trace are logged via l.
func Recovery(l logx.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if val := recover(); val != nil {
				l.Error("panic recovered",
					"error", val,
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
			}
		}()
		c.Next()
	}
}
