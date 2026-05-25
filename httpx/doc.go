// Package httpx provides a resilient HTTP client with production-safe defaults.
//
// The [Client] wraps [net/http.Client] with:
//   - Per-request timeout (default 30s)
//   - Automatic retries with exponential backoff + jitter (default 3 retries)
//   - Circuit breaker (trips after 5 consecutive failures, recovers after 30s)
//   - Optional OpenTelemetry tracing (span per request)
//   - Optional structured logging (one log line per completed request)
//
// Every default is overridable via functional options.
//
// # Quick start
//
//	client := httpx.New(
//	    httpx.WithTimeout(10*time.Second),
//	    httpx.WithRetries(3),
//	)
//
//	var result MyResponse
//	if err := client.GetJSON(ctx, "https://api.example.com/data", &result); err != nil {
//	    if errors.Is(err, httpx.ErrBreakerOpen) {
//	        // upstream is down; fail fast
//	    }
//	    return err
//	}
//
// # Default values
//
//	Timeout:               30s
//	DialTimeout:            5s
//	TLSTimeout:             5s
//	ResponseHeaderTimeout: 10s
//	IdleTimeout:           90s
//	MaxIdleConns:         100
//	MaxConnsPerHost:       10
//	Retries:                3
//	RetryOnStatus:  [502, 503, 504]
//	BreakerThreshold:       5 consecutive failures
//	BreakerInterval:       30s
package httpx
