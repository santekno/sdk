package logx

import (
	"context"
	"io"
	"log/slog"
)

// noopLogger discards everything.
type noopLogger struct{}

var defaultNoop Logger = &noopLogger{}

// Noop returns a Logger that discards all log lines. Useful for tests
// or for disabling logging at a call site.
func Noop() Logger { return defaultNoop }

func (n *noopLogger) Debug(string, ...any)               {}
func (n *noopLogger) Info(string, ...any)                {}
func (n *noopLogger) Warn(string, ...any)                {}
func (n *noopLogger) Error(string, ...any)               {}
func (n *noopLogger) With(...any) Logger                 { return n }
func (n *noopLogger) WithContext(context.Context) Logger { return n }
func (n *noopLogger) Slog() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}
