package logx_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/santekno/sdk/logx"
)

func TestNew_DefaultJSON(t *testing.T) {
	var buf bytes.Buffer
	l := logx.New(logx.WithOutput(&buf))
	l.Info("hello", "k", "v")

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, buf.String())
	}
	if got["msg"] != "hello" {
		t.Errorf("msg = %v, want hello", got["msg"])
	}
	if got["k"] != "v" {
		t.Errorf("k = %v, want v", got["k"])
	}
}

func TestNew_Console(t *testing.T) {
	var buf bytes.Buffer
	l := logx.New(logx.WithOutput(&buf), logx.WithFormat("console"))
	l.Info("hello")
	if !strings.Contains(buf.String(), "hello") {
		t.Errorf("output missing message: %q", buf.String())
	}
}

func TestLevels(t *testing.T) {
	var buf bytes.Buffer
	l := logx.New(logx.WithOutput(&buf), logx.WithLevel("warn"))
	l.Debug("debug-line")
	l.Info("info-line")
	l.Warn("warn-line")
	l.Error("error-line")

	out := buf.String()
	if strings.Contains(out, "debug-line") {
		t.Error("debug should be filtered out at warn level")
	}
	if strings.Contains(out, "info-line") {
		t.Error("info should be filtered out at warn level")
	}
	if !strings.Contains(out, "warn-line") {
		t.Error("warn-line missing")
	}
	if !strings.Contains(out, "error-line") {
		t.Error("error-line missing")
	}
}

func TestWith(t *testing.T) {
	var buf bytes.Buffer
	l := logx.New(logx.WithOutput(&buf)).With("service", "test")
	l.Info("hello")
	var got map[string]any
	_ = json.Unmarshal(buf.Bytes(), &got)
	if got["service"] != "test" {
		t.Errorf("service = %v, want test", got["service"])
	}
}

func TestFromContext_DefaultNoop(t *testing.T) {
	l := logx.FromContext(context.Background())
	if l == nil {
		t.Fatal("FromContext returned nil")
	}
	l.Info("should not panic")
}

func TestWithContextLogger_Roundtrip(t *testing.T) {
	var buf bytes.Buffer
	orig := logx.New(logx.WithOutput(&buf))
	ctx := logx.WithContextLogger(context.Background(), orig)
	back := logx.FromContext(ctx)
	back.Info("hello")
	if !strings.Contains(buf.String(), "hello") {
		t.Errorf("roundtrip logger did not write: %q", buf.String())
	}
}

func TestContextExtractor(t *testing.T) {
	type ctxKey struct{}
	logx.RegisterContextExtractor(func(ctx context.Context) (string, any, bool) {
		if v, ok := ctx.Value(ctxKey{}).(string); ok {
			return "request_id", v, true
		}
		return "", nil, false
	})

	var buf bytes.Buffer
	l := logx.New(logx.WithOutput(&buf))
	ctx := context.WithValue(context.Background(), ctxKey{}, "req-123")
	l.WithContext(ctx).Info("hello")

	var got map[string]any
	_ = json.Unmarshal(buf.Bytes(), &got)
	if got["request_id"] != "req-123" {
		t.Errorf("request_id = %v, want req-123", got["request_id"])
	}
}

func TestSlogAdapter(t *testing.T) {
	var buf bytes.Buffer
	l := logx.New(logx.WithOutput(&buf))
	s := l.Slog()
	if s == nil {
		t.Fatal("Slog() returned nil")
	}
	s.Info("via-slog")
	if !strings.Contains(buf.String(), "via-slog") {
		t.Errorf("slog adapter did not write: %q", buf.String())
	}
}

func TestNoop(t *testing.T) {
	n := logx.Noop()
	// All operations must be safe
	n.Debug("x")
	n.Info("x")
	n.Warn("x")
	n.Error("x")
	n.With("k", "v").Info("x")
	n.WithContext(context.Background()).Info("x")
	if n.Slog() == nil {
		t.Error("Noop().Slog() returned nil")
	}
}
