package httpx

import "time"

// Default configuration values for [Client].
// Every value is exported so users can reference them when building custom options.
const (
	DefaultTimeout               = 30 * time.Second
	DefaultDialTimeout           = 5 * time.Second
	DefaultTLSTimeout            = 5 * time.Second
	DefaultResponseHeaderTimeout = 10 * time.Second
	DefaultIdleTimeout           = 90 * time.Second
	DefaultMaxIdleConns          = 100
	DefaultMaxConnsPerHost       = 10
	DefaultRetries               = 3
	DefaultBreakerThreshold      = 5
	DefaultBreakerInterval       = 30 * time.Second
	DefaultBreakerTimeout        = 30 * time.Second
)
