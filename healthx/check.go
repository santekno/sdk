package healthx

import (
	"context"
	"errors"
	"net/http"
	"time"
)

// Check is a single health probe. It returns nil if the dependency is healthy.
type Check func(ctx context.Context) error

// NamedCheck attaches a human-readable name to a Check.
type NamedCheck struct {
	Name  string
	Check Check
}

// Handler exposes liveness and readiness probes for HTTP serving.
type Handler struct {
	Liveness  []NamedCheck
	Readiness []NamedCheck
	// Timeout bounds the per-check execution time. Defaults to 2s if zero.
	Timeout time.Duration
}

// AlwaysOK returns a Check that always succeeds.
func AlwaysOK() Check {
	return func(context.Context) error { return nil }
}

// AlwaysFail returns a Check that always fails. Useful for testing.
func AlwaysFail(msg string) Check {
	return func(context.Context) error { return errors.New(msg) }
}

// PingHTTP returns a Check that performs a GET on url and expects a 2xx response.
func PingHTTP(url string) Check {
	return func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return errors.New("healthx: ping " + url + " returned non-2xx")
		}
		return nil
	}
}

// Pinger is the minimal interface for objects that can be health-checked
// via a Ping method (sql.DB, redis.Client all satisfy this shape).
type Pinger interface {
	PingContext(ctx context.Context) error
}

// PingPinger returns a Check that calls p.PingContext.
func PingPinger(p Pinger) Check {
	return func(ctx context.Context) error { return p.PingContext(ctx) }
}
