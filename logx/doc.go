// Package logx provides structured logging with a pluggable backend.
//
// The [Logger] interface is the central abstraction — any SDK package that
// accepts logx.Logger can be wired to any backend implementation. The
// default backend uses stdlib log/slog; a uber-go/zap backend is planned.
//
//	logger := logx.New(logx.WithLevel("info"), logx.WithFormat("json"))
//	logger.Info("hello", "user_id", "u-1")
//
// Use WithContext to propagate OTel trace identifiers into log lines:
//
//	logger.WithContext(ctx).Info("processing", "amount", 1000)
package logx
