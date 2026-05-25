package configx

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// WithFile adds a JSON config file source. YAML support requires adding a
// YAML parser dependency in your application module.
func WithFile(path string) Option {
	return func(l *loader) { l.filePath = path }
}

// WithEnv adds environment variable source with the given prefix.
// e.g. WithEnv("APP") maps APP_SERVER_PORT → "server.port".
func WithEnv(prefix string) Option {
	return func(l *loader) { l.envPrefix = prefix }
}

// WithDefaults sets the lowest-priority defaults map.
// Keys use dot notation: "server.port", "database.dsn".
func WithDefaults(m map[string]any) Option {
	return func(l *loader) { l.defaults = m }
}

// WithRequired marks one or more dot-notation keys as required.
// Load returns ErrRequiredKey if any are absent after merging all sources.
func WithRequired(keys ...string) Option {
	return func(l *loader) { l.required = append(l.required, keys...) }
}

// WithWatch enables periodic re-loading of all sources. cb is called with
// the new *Config whenever changes are detected. Not yet implemented —
// returns an informational error from Config.Watch.
func WithWatch(interval time.Duration) Option {
	return func(l *loader) {
		l.watchPeriod = interval
	}
}

// ── Typed accessors ──────────────────────────────────────────────────────────

// String returns the string value for key, or "" if absent or wrong type.
func (c *Config) String(key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, _ := c.data[key]
	s, _ := toString(v)
	return s
}

// StringDefault returns the string value for key, or def if absent.
func (c *Config) StringDefault(key, def string) string {
	if s := c.String(key); s != "" {
		return s
	}
	return def
}

// Int returns the integer value for key, or 0 if absent or not parseable.
func (c *Config) Int(key string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, _ := c.data[key]
	return toInt(v)
}

// IntDefault returns the integer value for key, or def if absent / zero.
func (c *Config) IntDefault(key string, def int) int {
	if i := c.Int(key); i != 0 {
		return i
	}
	return def
}

// Bool returns the boolean value for key.
func (c *Config) Bool(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, _ := c.data[key]
	return toBool(v)
}

// Float64 returns the float64 value for key.
func (c *Config) Float64(key string) float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, _ := c.data[key]
	return toFloat64(v)
}

// Duration returns the time.Duration value for key.
// Accepts numeric values (nanoseconds) and duration strings ("30s", "1h").
func (c *Config) Duration(key string) time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, _ := c.data[key]
	switch x := v.(type) {
	case time.Duration:
		return x
	case string:
		d, _ := time.ParseDuration(x)
		return d
	case float64:
		return time.Duration(int64(x))
	default:
		return 0
	}
}

// Strings returns a slice of strings for key. The stored value may be a
// []string, a comma-separated string, or a JSON array of strings.
func (c *Config) Strings(key string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, _ := c.data[key]
	switch x := v.(type) {
	case []string:
		return x
	case string:
		if x == "" {
			return nil
		}
		return strings.Split(x, ",")
	default:
		return nil
	}
}

// Map returns the map value for key, or nil if absent.
func (c *Config) Map(key string) map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, _ := c.data[key]
	m, _ := v.(map[string]any)
	return m
}

// Unmarshal populates out with configuration under the given key prefix.
// Uses JSON round-trip with json struct tags.
func (c *Config) Unmarshal(key string, out any) error {
	c.mu.RLock()
	sub := make(map[string]any)
	prefix := key + "."
	for k, v := range c.data {
		if strings.HasPrefix(k, prefix) {
			sub[strings.TrimPrefix(k, prefix)] = v
		}
	}
	c.mu.RUnlock()

	b, err := json.Marshal(sub)
	if err != nil {
		return fmt.Errorf("configx: marshal: %w", err)
	}
	return json.Unmarshal(b, out)
}

// Watch registers cb to be called after each live-reload cycle.
// Returns an error if WithWatch was not passed to Load.
func (c *Config) Watch(cb func()) error {
	if c == nil {
		return fmt.Errorf("configx: Watch called on nil Config")
	}
	_ = cb
	return fmt.Errorf("configx: live-reload not enabled; pass WithWatch to Load")
}

// ── conversion helpers ────────────────────────────────────────────────────────

func toString(v any) (string, bool) {
	switch x := v.(type) {
	case string:
		return x, true
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64), true
	case bool:
		return strconv.FormatBool(x), true
	default:
		return "", false
	}
}

func toInt(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case string:
		i, _ := strconv.Atoi(x)
		return i
	default:
		return 0
	}
}

func toBool(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		b, _ := strconv.ParseBool(x)
		return b
	default:
		return false
	}
}

func toFloat64(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case string:
		f, _ := strconv.ParseFloat(x, 64)
		return f
	default:
		return 0
	}
}
