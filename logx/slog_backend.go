package logx

import (
	"context"
	"log/slog"
)

// slogLogger is the stdlib log/slog-backed implementation of Logger.
type slogLogger struct {
	s   *slog.Logger
	cfg *config
}

// New returns a Logger configured with opts. Never returns nil.
func New(opts ...Option) Logger {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	level := parseLevel(cfg.Level)
	handlerOpts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.Caller,
	}

	var h slog.Handler
	if cfg.Format == "console" || cfg.Format == "text" {
		h = slog.NewTextHandler(cfg.Output, handlerOpts)
	} else {
		h = slog.NewJSONHandler(cfg.Output, handlerOpts)
	}

	s := slog.New(h)
	if len(cfg.Fields) > 0 {
		s = s.With(cfg.Fields...)
	}
	return &slogLogger{s: s, cfg: cfg}
}

func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func (l *slogLogger) Debug(msg string, kv ...any) { l.s.Debug(msg, kv...) }
func (l *slogLogger) Info(msg string, kv ...any)  { l.s.Info(msg, kv...) }
func (l *slogLogger) Warn(msg string, kv ...any)  { l.s.Warn(msg, kv...) }
func (l *slogLogger) Error(msg string, kv ...any) { l.s.Error(msg, kv...) }

func (l *slogLogger) With(kv ...any) Logger {
	return &slogLogger{s: l.s.With(kv...), cfg: l.cfg}
}

func (l *slogLogger) WithContext(ctx context.Context) Logger {
	fields := extractContextFields(ctx)
	if len(fields) == 0 {
		return l
	}
	return &slogLogger{s: l.s.With(fields...), cfg: l.cfg}
}

func (l *slogLogger) Slog() *slog.Logger { return l.s }
