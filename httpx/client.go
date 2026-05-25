package httpx

import (
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Client is a resilient HTTP client built on top of [net/http.Client].
// It provides timeouts, retries with exponential backoff, a circuit breaker,
// and hooks for structured logging and OpenTelemetry tracing.
//
// Client is safe for concurrent use by multiple goroutines.
type Client struct {
	httpClient *http.Client
	cfg        *config

	// circuit breaker state
	breakerMu       sync.Mutex
	consecutiveFail uint32
	breakerState    atomic.Uint32 // 0=closed, 1=half-open, 2=open
	breakerOpenedAt time.Time
}

// New returns a new resilient [Client] configured with the supplied options.
// Default values are applied first, then opts in order. New never returns nil
// and never panics.
func New(opts ...Option) *Client {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		transport := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   cfg.DialTimeout,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          cfg.MaxIdleConns,
			MaxConnsPerHost:       cfg.MaxConnsPerHost,
			MaxIdleConnsPerHost:   cfg.MaxConnsPerHost,
			IdleConnTimeout:       cfg.IdleTimeout,
			TLSHandshakeTimeout:   cfg.TLSTimeout,
			ResponseHeaderTimeout: cfg.ResponseHeaderTimeout,
			ExpectContinueTimeout: 1 * time.Second,
		}
		httpClient = &http.Client{
			Transport: transport,
			Timeout:   cfg.Timeout,
		}
	}

	return &Client{
		httpClient: httpClient,
		cfg:        cfg,
	}
}
