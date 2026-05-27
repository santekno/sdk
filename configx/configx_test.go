package configx_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/santekno/sdk/configx"
)

// writeJSON writes a JSON file and returns the path.
func writeJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoad_Defaults(t *testing.T) {
	cfg, err := configx.Load(
		configx.WithDefaults(map[string]any{
			"server.port": 8080,
			"app.name":    "myapp",
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Int("server.port") != 8080 {
		t.Errorf("server.port = %d, want 8080", cfg.Int("server.port"))
	}
	if cfg.String("app.name") != "myapp" {
		t.Errorf("app.name = %q, want myapp", cfg.String("app.name"))
	}
}

func TestLoad_File(t *testing.T) {
	path := writeJSON(t, map[string]any{
		"database": map[string]any{"dsn": "postgres://localhost/test"},
		"workers":  3,
	})

	cfg, err := configx.Load(configx.WithFile(path))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.String("database.dsn") != "postgres://localhost/test" {
		t.Errorf("database.dsn = %q", cfg.String("database.dsn"))
	}
	if cfg.Int("workers") != 3 {
		t.Errorf("workers = %d, want 3", cfg.Int("workers"))
	}
}

func TestLoad_Env(t *testing.T) {
	t.Setenv("MYAPP_SERVER_PORT", "9090")
	t.Setenv("MYAPP_DATABASE_DSN", "postgres://env/test")

	cfg, err := configx.Load(configx.WithEnv("MYAPP"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.String("server.port") != "9090" {
		t.Errorf("server.port = %q, want 9090", cfg.String("server.port"))
	}
	if cfg.String("database.dsn") != "postgres://env/test" {
		t.Errorf("database.dsn = %q", cfg.String("database.dsn"))
	}
}

func TestLoad_Priority(t *testing.T) {
	// env > file > defaults
	path := writeJSON(t, map[string]any{"port": 8081})
	t.Setenv("PRIO_PORT", "8082")

	cfg, err := configx.Load(
		configx.WithDefaults(map[string]any{"port": 8080}),
		configx.WithFile(path),
		configx.WithEnv("PRIO"),
	)
	if err != nil {
		t.Fatal(err)
	}
	// env wins
	if cfg.String("port") != "8082" {
		t.Errorf("port = %q, want 8082 (env priority)", cfg.String("port"))
	}
}

func TestLoad_Required_Missing(t *testing.T) {
	_, err := configx.Load(configx.WithRequired("database.dsn"))
	if err == nil {
		t.Fatal("expected error for missing required key")
	}
}

func TestLoad_Required_Present(t *testing.T) {
	cfg, err := configx.Load(
		configx.WithDefaults(map[string]any{"database.dsn": "sqlite://:memory:"}),
		configx.WithRequired("database.dsn"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.String("database.dsn") == "" {
		t.Error("database.dsn should not be empty")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := configx.Load(configx.WithFile("/nonexistent/path/config.json"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestConfig_StringDefault(t *testing.T) {
	cfg, _ := configx.Load()
	if got := cfg.StringDefault("missing", "fallback"); got != "fallback" {
		t.Errorf("StringDefault = %q, want fallback", got)
	}
}

func TestConfig_IntDefault(t *testing.T) {
	cfg, _ := configx.Load()
	if got := cfg.IntDefault("missing", 42); got != 42 {
		t.Errorf("IntDefault = %d, want 42", got)
	}
}

func TestConfig_Bool(t *testing.T) {
	cfg, _ := configx.Load(
		configx.WithDefaults(map[string]any{
			"feature.enabled": true,
			"feature.str":     "true",
		}),
	)
	if !cfg.Bool("feature.enabled") {
		t.Error("feature.enabled should be true")
	}
	if !cfg.Bool("feature.str") {
		t.Error("feature.str should be true")
	}
}

func TestConfig_Float64(t *testing.T) {
	cfg, _ := configx.Load(
		configx.WithDefaults(map[string]any{"ratio": 0.75}),
	)
	if cfg.Float64("ratio") != 0.75 {
		t.Errorf("ratio = %f, want 0.75", cfg.Float64("ratio"))
	}
}

func TestConfig_Duration(t *testing.T) {
	cfg, _ := configx.Load(
		configx.WithDefaults(map[string]any{
			"timeout": "30s",
		}),
	)
	if cfg.Duration("timeout") != 30*time.Second {
		t.Errorf("timeout = %v, want 30s", cfg.Duration("timeout"))
	}
}

func TestConfig_Strings(t *testing.T) {
	cfg, _ := configx.Load(
		configx.WithDefaults(map[string]any{"hosts": "a,b,c"}),
	)
	got := cfg.Strings("hosts")
	if len(got) != 3 || got[0] != "a" || got[2] != "c" {
		t.Errorf("Strings = %v, want [a b c]", got)
	}
}

func TestConfig_Unmarshal(t *testing.T) {
	type DBConfig struct {
		DSN     string `json:"dsn"`
		MaxConn int    `json:"maxconn"`
	}

	path := writeJSON(t, map[string]any{
		"database": map[string]any{
			"dsn":     "postgres://localhost/test",
			"maxconn": 10,
		},
	})
	cfg, err := configx.Load(configx.WithFile(path))
	if err != nil {
		t.Fatal(err)
	}

	var db DBConfig
	if err := cfg.Unmarshal("database", &db); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if db.DSN != "postgres://localhost/test" {
		t.Errorf("DSN = %q", db.DSN)
	}
	if db.MaxConn != 10 {
		t.Errorf("MaxConn = %d, want 10", db.MaxConn)
	}
}

func TestConfig_Watch_NotEnabled(t *testing.T) {
	cfg, _ := configx.Load()
	if err := cfg.Watch(func() {}); err == nil {
		t.Error("expected error when WithWatch not passed")
	}
}

func TestWithWatch(t *testing.T) {
	// WithWatch sets the period; Watch still returns error (not yet implemented).
	cfg, err := configx.Load(configx.WithWatch(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	// Watch currently always returns an error (stub implementation).
	if err := cfg.Watch(func() {}); err == nil {
		t.Error("expected Watch to return not-enabled error")
	}
}
