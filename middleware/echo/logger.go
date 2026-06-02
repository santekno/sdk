package echo

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/santekno/sdk/ctxx"
	"github.com/santekno/sdk/logx"
)

// Logger returns an Echo middleware that logs each request with method, path,
// status, duration, and request_id extracted from the request context.
//
// Mirrors github.com/santekno/sdk/middleware/gin.Logger.
func Logger(l logx.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			req := c.Request()
			path := req.URL.Path
			method := req.Method

			err := next(c)

			l.Info("http request",
				"method", method,
				"path", path,
				"status", c.Response().Status,
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", ctxx.RequestID(req.Context()),
				"client_ip", c.RealIP(),
			)
			return err
		}
	}
}
