// Package tracex provides a vendor-neutral tracing interface that integrates
// with OpenTelemetry without forcing OTel as a dependency.
//
// Users who want OpenTelemetry can adapt their go.opentelemetry.io/otel/trace
// implementation to the tracex.Tracer interface. The default global tracer
// is a no-op so the SDK is usable without any tracing setup.
//
//	tr := tracex.Global()
//	ctx, span := tr.Start(ctx, "my-operation")
//	defer span.End()
package tracex
