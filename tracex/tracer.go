package tracex

import (
	"context"
	"sync/atomic"
)

// Tracer creates spans. The implementation may be a no-op, an OTel wrapper,
// or any other tracing backend.
type Tracer interface {
	Start(ctx context.Context, name string, opts ...SpanStartOption) (context.Context, Span)
}

// Span represents a single unit of work within a trace.
type Span interface {
	// End completes the span. Calling End multiple times is a no-op.
	End()
	// SetAttribute sets a key-value attribute on the span.
	SetAttribute(key string, value any)
	// SetStatus records the span's status: "ok", "error", or "unset".
	SetStatus(code string, description string)
	// RecordError records an error on the span. Implementations SHOULD
	// also set the status to "error".
	RecordError(err error)
}

// SpanStartOption is a placeholder for span start options.
// It is intentionally opaque to remain compatible with OTel options.
type SpanStartOption any

// noopTracer satisfies Tracer with zero overhead.
type noopTracer struct{}

func (noopTracer) Start(ctx context.Context, _ string, _ ...SpanStartOption) (context.Context, Span) {
	return ctx, noopSpan{}
}

type noopSpan struct{}

func (noopSpan) End()                     {}
func (noopSpan) SetAttribute(string, any) {}
func (noopSpan) SetStatus(string, string) {}
func (noopSpan) RecordError(error)        {}

// NoopTracer returns a Tracer that records nothing.
func NoopTracer() Tracer { return noopTracer{} }

// globalTracer holds the application-wide default tracer.
var globalTracer atomic.Value

func init() { globalTracer.Store(Tracer(noopTracer{})) }

// Global returns the application-wide tracer. Defaults to a no-op tracer.
func Global() Tracer { return globalTracer.Load().(Tracer) }

// SetGlobal sets the application-wide tracer. Pass nil to reset to the no-op.
func SetGlobal(t Tracer) {
	if t == nil {
		t = noopTracer{}
	}
	globalTracer.Store(t)
}
