package logx

import "context"

// ContextExtractor extracts a key-value pair from ctx for inclusion in log
// lines. Returns (key, value, ok=true) if the field is present.
//
// Register custom extractors at package init time to support project-specific
// context keys.
type ContextExtractor func(ctx context.Context) (key string, value any, ok bool)

// contextExtractors is the registered list of context-to-log-field extractors.
// Default extractors look up request_id and user_id via well-known context keys.
var contextExtractors []ContextExtractor

// RegisterContextExtractor adds a global extractor invoked by WithContext.
// Safe to call at init time. Not safe for concurrent calls.
func RegisterContextExtractor(fn ContextExtractor) {
	contextExtractors = append(contextExtractors, fn)
}

func extractContextFields(ctx context.Context) []any {
	if ctx == nil {
		return nil
	}
	var out []any
	for _, fn := range contextExtractors {
		if k, v, ok := fn(ctx); ok {
			out = append(out, k, v)
		}
	}
	return out
}

// loggerCtxKey is the context key under which a Logger may be stored.
type loggerCtxKey struct{}

// FromContext returns the Logger attached to ctx, or a no-op logger if none.
func FromContext(ctx context.Context) Logger {
	if ctx == nil {
		return defaultNoop
	}
	if l, ok := ctx.Value(loggerCtxKey{}).(Logger); ok {
		return l
	}
	return defaultNoop
}

// WithContextLogger returns a new context with l attached.
func WithContextLogger(ctx context.Context, l Logger) context.Context {
	return context.WithValue(ctx, loggerCtxKey{}, l)
}
