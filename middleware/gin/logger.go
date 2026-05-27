package gin

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/santekno/sdk/ctxx"
	"github.com/santekno/sdk/logx"
)

// Logger returns a Gin middleware that logs each request with method, path,
// status, duration, and trace_id extracted from the request context.
func Logger(l logx.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		l.Info("http request",
			"method", method,
			"path", path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", ctxx.RequestID(c.Request.Context()),
			"client_ip", c.ClientIP(),
		)
	}
}
