package gin

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
	"github.com/santekno/sdk/ctxx"
)

// Tracing returns a Gin middleware that propagates or generates a request ID
// stored in the context via ctxx.WithRequestID for downstream correlation.
//
// Checks X-Request-ID and traceparent headers; generates a random ID if absent.
// For full OTel integration, wire your OTel tracer into httpx and tracex.
func Tracing() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = c.GetHeader("traceparent")
		}
		if requestID == "" {
			requestID = randomHex(8)
		}

		c.Header("X-Request-ID", requestID)
		ctx := ctxx.WithRequestID(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
