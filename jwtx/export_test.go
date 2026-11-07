package jwtx

import "time"

// SetClock replaces the client's clock. Call it before the first Lookup.
func (c *JWKSClient) SetClock(now func() time.Time) { c.now = now }

// SetTimeout replaces the fetch timeout. Call it before the first Lookup.
func (c *JWKSClient) SetTimeout(d time.Duration) { c.timeout = d }

// SetNow overrides the package clock and returns a function that restores it.
func SetNow(now func() time.Time) (restore func()) {
	old := nowFunc
	nowFunc = now
	return func() { nowFunc = old }
}
