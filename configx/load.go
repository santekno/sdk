package configx

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// ErrFileNotFound is returned when a config file path does not exist.
var ErrFileNotFound = errors.New("configx: file not found")

// ErrRequiredKey is returned when one or more required keys are missing after loading.
var ErrRequiredKey = errors.New("configx: required key missing")

// Config holds the merged configuration map and provides typed accessors.
// Safe for concurrent reads after Load returns.
type Config struct {
	mu   sync.RWMutex
	data map[string]any
}

// loader holds the pending options before Load is called.
type loader struct {
	defaults    map[string]any
	filePath    string
	envPrefix   string
	required    []string
	watchFn     func()
	watchPeriod time.Duration
}

// Option is a functional option for Load.
type Option func(*loader)

// Load merges all configured sources and returns the resulting *Config.
// Returns an error if a required key is missing or a file cannot be read.
func Load(opts ...Option) (*Config, error) {
	l := &loader{}
	for _, o := range opts {
		o(l)
	}

	merged := make(map[string]any)

	// Priority 3 (lowest): defaults.
	for k, v := range l.defaults {
		flatSet(merged, k, v)
	}

	// Priority 2: file.
	if l.filePath != "" {
		fm, err := loadFile(l.filePath)
		if err != nil {
			return nil, err
		}
		for k, v := range flatten(fm, "") {
			flatSet(merged, k, v)
		}
	}

	// Priority 1 (highest): environment variables.
	if l.envPrefix != "" {
		prefix := strings.ToUpper(l.envPrefix) + "_"
		for _, env := range os.Environ() {
			kv := strings.SplitN(env, "=", 2)
			if len(kv) != 2 {
				continue
			}
			if !strings.HasPrefix(kv[0], prefix) {
				continue
			}
			// Strip prefix, lowercase, replace _ with .
			key := strings.ToLower(strings.TrimPrefix(kv[0], prefix))
			key = strings.ReplaceAll(key, "_", ".")
			flatSet(merged, key, kv[1])
		}
	}

	// Validate required keys.
	var missing []string
	for _, req := range l.required {
		if _, ok := merged[req]; !ok {
			missing = append(missing, req)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrRequiredKey, strings.Join(missing, ", "))
	}

	return &Config{data: merged}, nil
}

// loadFile reads and JSON-decodes the file at path.
func loadFile(path string) (map[string]any, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrFileNotFound, path)
		}
		return nil, fmt.Errorf("configx: read %s: %w", path, err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("configx: parse %s: %w", path, err)
	}
	return m, nil
}

// flatten converts a nested map to a flat map with dot-separated keys.
func flatten(m map[string]any, prefix string) map[string]any {
	out := make(map[string]any)
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		if sub, ok := v.(map[string]any); ok {
			for fk, fv := range flatten(sub, key) {
				out[fk] = fv
			}
		} else {
			out[key] = v
		}
	}
	return out
}

// flatSet stores value at the dot-separated key.
func flatSet(m map[string]any, key string, value any) {
	m[key] = value
}
