package echo

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/labstack/echo/v4"
	"github.com/santekno/sdk/ctxx"
)

// Tracing returns an Echo middleware that propagates or generates a request ID
// stored in the context via ctxx.WithRequestID for downstream correlation.
//
// Checks X-Request-ID and traceparent headers; generates a random ID if absent.
// The resolved request ID is echoed back in the X-Request-ID response header.
// For full OTel integration, wire your OTel tracer into httpx and tracex.
//
// Mirrors github.com/santekno/sdk/middleware/gin.Tracing.
func Tracing() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			requestID := req.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = req.Header.Get("traceparent")
			}
			if requestID == "" {
				requestID = randomHex(8)
			}

			c.Response().Header().Set("X-Request-ID", requestID)
			ctx := ctxx.WithRequestID(req.Context(), requestID)
			c.SetRequest(req.WithContext(ctx))
			return next(c)
		}
	}
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
