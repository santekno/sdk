package httpx

import (
	"net/http"
	"time"
)

// config holds all configuration for a [Client].
// Users interact with it only through [Option] functions.
type config struct {
	BaseURL               string
	Timeout               time.Duration
	DialTimeout           time.Duration
	TLSTimeout            time.Duration
	ResponseHeaderTimeout time.Duration
	IdleTimeout           time.Duration
	MaxIdleConns          int
	MaxConnsPerHost       int

	Retries        int
	RetryOnStatus  []int
	RetryOnMethods map[string]struct{}

	BreakerName      string
	BreakerThreshold uint32
	BreakerInterval  time.Duration
	BreakerTimeout   time.Duration

	Logger Logger
	// Tracer is stored as any to avoid importing go.opentelemetry.io/otel
	// in the base package when tracing is not used.
	Tracer any

	DefaultHeaders http.Header
	BearerToken    string
	APIKey         string

	// HTTPClient allows advanced users to supply their own *http.Client.
	HTTPClient *http.Client
}

// Option is a functional option for configuring a [Client].
type Option func(*config)

func defaultConfig() *config {
	return &config{
		Timeout:               DefaultTimeout,
		DialTimeout:           DefaultDialTimeout,
		TLSTimeout:            DefaultTLSTimeout,
		ResponseHeaderTimeout: DefaultResponseHeaderTimeout,
		IdleTimeout:           DefaultIdleTimeout,
		MaxIdleConns:          DefaultMaxIdleConns,
		MaxConnsPerHost:       DefaultMaxConnsPerHost,
		Retries:               DefaultRetries,
		RetryOnStatus:         []int{502, 503, 504},
		RetryOnMethods: map[string]struct{}{
			http.MethodGet:    {},
			http.MethodHead:   {},
			http.MethodPut:    {},
			http.MethodDelete: {},
		},
		BreakerThreshold: DefaultBreakerThreshold,
		BreakerInterval:  DefaultBreakerInterval,
		BreakerTimeout:   DefaultBreakerTimeout,
		DefaultHeaders:   make(http.Header),
	}
}
