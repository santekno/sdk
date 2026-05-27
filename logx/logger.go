package logx

import (
	"context"
	"log/slog"
)

// Logger is the structured logging interface used across the Santekno Go SDK.
//
// Implementations MUST be safe for concurrent use.
type Logger interface {
	Debug(msg string, kv ...any)
	Info(msg string, kv ...any)
	Warn(msg string, kv ...any)
	Error(msg string, kv ...any)

	// With returns a new Logger that includes the supplied key-value pairs
	// in every emitted log line.
	With(kv ...any) Logger

	// WithContext returns a new Logger that extracts well-known fields from ctx
	// (trace_id, span_id, request_id, user_id) and includes them in every line.
	WithContext(ctx context.Context) Logger

	// Slog returns a *slog.Logger that delegates to this Logger's backend.
	Slog() *slog.Logger
}
