package httpx

import (
	"net/http"
	"time"
)

// WithBaseURL sets a base URL that is prepended to relative request paths.
func WithBaseURL(u string) Option { return func(c *config) { c.BaseURL = u } }

// WithTimeout sets the per-request timeout (default [DefaultTimeout]).
func WithTimeout(d time.Duration) Option { return func(c *config) { c.Timeout = d } }

// WithDialTimeout sets the TCP dial timeout (default [DefaultDialTimeout]).
func WithDialTimeout(d time.Duration) Option { return func(c *config) { c.DialTimeout = d } }

// WithTLSTimeout sets the TLS handshake timeout (default [DefaultTLSTimeout]).
func WithTLSTimeout(d time.Duration) Option { return func(c *config) { c.TLSTimeout = d } }

// WithResponseHeaderTimeout sets the response-header read timeout
// (default [DefaultResponseHeaderTimeout]).
func WithResponseHeaderTimeout(d time.Duration) Option {
	return func(c *config) { c.ResponseHeaderTimeout = d }
}

// WithIdleTimeout sets the idle connection timeout (default [DefaultIdleTimeout]).
func WithIdleTimeout(d time.Duration) Option { return func(c *config) { c.IdleTimeout = d } }

// WithMaxIdleConns sets the maximum number of idle connections across all hosts
// (default [DefaultMaxIdleConns]).
func WithMaxIdleConns(n int) Option { return func(c *config) { c.MaxIdleConns = n } }

// WithMaxConnsPerHost sets the maximum number of connections per host
// (default [DefaultMaxConnsPerHost]).
func WithMaxConnsPerHost(n int) Option { return func(c *config) { c.MaxConnsPerHost = n } }

// WithRetries sets the number of retry attempts (default [DefaultRetries]).
func WithRetries(n int) Option { return func(c *config) { c.Retries = n } }

// WithRetryOnStatus sets the HTTP status codes that trigger a retry
// (default: 502, 503, 504).
func WithRetryOnStatus(codes ...int) Option {
	return func(c *config) { c.RetryOnStatus = codes }
}

// WithRetryOnMethods sets the HTTP methods that are eligible for retry.
// Default: GET, HEAD, PUT, DELETE. Use this to add POST (idempotent endpoints).
func WithRetryOnMethods(methods ...string) Option {
	return func(c *config) {
		m := make(map[string]struct{}, len(methods))
		for _, method := range methods {
			m[method] = struct{}{}
		}
		c.RetryOnMethods = m
	}
}

// WithCircuitBreaker configures the circuit breaker.
// name identifies the breaker for logging and metrics.
func WithCircuitBreaker(name string, threshold uint32, interval, timeout time.Duration) Option {
	return func(c *config) {
		c.BreakerName = name
		c.BreakerThreshold = threshold
		c.BreakerInterval = interval
		c.BreakerTimeout = timeout
	}
}

// WithLogger sets the structured logger for request logging.
func WithLogger(l Logger) Option { return func(c *config) { c.Logger = l } }

// WithTracer sets the OpenTelemetry tracer. Pass a trace.Tracer value.
// The tracer is stored as any to avoid importing OTel in users who don't use it.
func WithTracer(t any) Option { return func(c *config) { c.Tracer = t } }

// WithDefaultHeader adds a default HTTP header sent with every request.
func WithDefaultHeader(key, value string) Option {
	return func(c *config) { c.DefaultHeaders.Set(key, value) }
}

// WithBearerToken sets an Authorization: Bearer token sent with every request.
func WithBearerToken(token string) Option { return func(c *config) { c.BearerToken = token } }

// WithAPIKey sets an Authorization: ApiKey header sent with every request.
func WithAPIKey(key string) Option { return func(c *config) { c.APIKey = key } }

// WithHTTPClient replaces the underlying *http.Client entirely.
// Use this for advanced transport configuration.
func WithHTTPClient(hc *http.Client) Option { return func(c *config) { c.HTTPClient = hc } }
