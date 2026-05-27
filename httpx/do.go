package httpx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

const (
	breakerClosed   = uint32(0)
	breakerHalfOpen = uint32(1)
	breakerOpen     = uint32(2)
)

// Do executes the supplied HTTP request with the configured retry, backoff,
// and circuit-breaker policy.
//
// On success, the caller MUST close resp.Body. On error, the body is already closed.
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	if !c.breakerAllow() {
		return nil, fmt.Errorf("%w", ErrBreakerOpen)
	}

	c.applyDefaultHeaders(req)

	method := strings.ToUpper(req.Method)
	_, retryable := c.cfg.RetryOnMethods[method]

	var lastErr error
	maxAttempts := 1
	if retryable {
		maxAttempts = c.cfg.Retries + 1
	}

	start := time.Now()

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			d := c.backoffDuration(attempt)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(d):
			}
		}

		reqWithCtx := req.WithContext(ctx)
		resp, err := c.httpClient.Do(reqWithCtx)

		if err != nil {
			lastErr = err
			if isTimeoutErr(err) {
				lastErr = fmt.Errorf("%w: %v", ErrTimeout, err)
			}
			c.breakerRecord(false)
			continue
		}

		if c.shouldRetryStatus(resp.StatusCode) && attempt+1 < maxAttempts {
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("%w: status %d", ErrNon2xx, resp.StatusCode)
			c.breakerRecord(false)
			continue
		}

		c.breakerRecord(resp.StatusCode < 500)
		c.log("info", "http request completed",
			"method", method,
			"url", req.URL.String(),
			"status", resp.StatusCode,
			"duration_ms", time.Since(start).Milliseconds(),
			"retry_count", attempt,
		)

		return resp, nil
	}

	c.log("error", "http request failed after retries",
		"method", method,
		"url", req.URL.String(),
		"duration_ms", time.Since(start).Milliseconds(),
		"error", lastErr,
	)

	if lastErr == nil {
		lastErr = ErrMaxRetries
	}
	return nil, fmt.Errorf("%w: %v", ErrMaxRetries, lastErr)
}

func (c *Client) applyDefaultHeaders(req *http.Request) {
	for k, vv := range c.cfg.DefaultHeaders {
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}
	if c.cfg.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.BearerToken)
	} else if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "ApiKey "+c.cfg.APIKey)
	}
}

func (c *Client) shouldRetryStatus(status int) bool {
	for _, s := range c.cfg.RetryOnStatus {
		if s == status {
			return true
		}
	}
	return false
}

func (c *Client) backoffDuration(attempt int) time.Duration {
	// Exponential backoff with full jitter: base * 2^(attempt-1) randomized.
	base := 100 * time.Millisecond
	max := 10 * time.Second

	d := base * (1 << uint(attempt-1))
	if d > max {
		d = max
	}
	jitter := time.Duration(rand.Int63n(int64(d))) // #nosec G404 -- non-cryptographic jitter is intentional
	return jitter
}

func (c *Client) breakerAllow() bool {
	state := c.breakerState.Load()
	if state == breakerOpen {
		c.breakerMu.Lock()
		defer c.breakerMu.Unlock()
		if time.Since(c.breakerOpenedAt) > c.cfg.BreakerTimeout {
			c.breakerState.Store(breakerHalfOpen)
			return true
		}
		return false
	}
	return true
}

func (c *Client) breakerRecord(success bool) {
	c.breakerMu.Lock()
	defer c.breakerMu.Unlock()

	if success {
		c.consecutiveFail = 0
		c.breakerState.Store(breakerClosed)
		return
	}

	c.consecutiveFail++
	if c.consecutiveFail >= c.cfg.BreakerThreshold {
		c.breakerState.Store(breakerOpen)
		c.breakerOpenedAt = time.Now()
	}
}

func (c *Client) log(level, msg string, kv ...any) {
	if c.cfg.Logger == nil {
		return
	}
	switch level {
	case "debug":
		c.cfg.Logger.Debug(msg, kv...)
	case "warn":
		c.cfg.Logger.Warn(msg, kv...)
	case "error":
		c.cfg.Logger.Error(msg, kv...)
	default:
		c.cfg.Logger.Info(msg, kv...)
	}
}

// isTimeoutErr reports whether err represents a timeout (deadline exceeded
// or a net.Error with Timeout() == true).
func isTimeoutErr(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var nerr interface{ Timeout() bool }
	if errors.As(err, &nerr) && nerr.Timeout() {
		return true
	}
	return false
}

// closeBody drains and closes resp.Body so the connection can be reused.
func closeBody(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}
}
