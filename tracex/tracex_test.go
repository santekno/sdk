package tracex_test

import (
	"context"
	"errors"
	"testing"

	"github.com/santekno/sdk/tracex"
)

func TestNoopTracer(t *testing.T) {
	tr := tracex.NoopTracer()
	ctx, span := tr.Start(context.Background(), "op")
	if ctx == nil || span == nil {
		t.Fatal("Start returned nil")
	}
	// All span operations are no-ops; must not panic.
	span.SetAttribute("k", "v")
	span.SetStatus("ok", "")
	span.RecordError(errors.New("boom"))
	span.End()
	span.End() // multiple End calls are safe
}

func TestGlobal(t *testing.T) {
	if tracex.Global() == nil {
		t.Fatal("Global returned nil")
	}
	custom := tracex.NoopTracer()
	tracex.SetGlobal(custom)
	if tracex.Global() != custom {
		t.Error("SetGlobal did not take effect")
	}
	// Setting nil should reset to default no-op (not nil).
	tracex.SetGlobal(nil)
	if tracex.Global() == nil {
		t.Error("Global returned nil after SetGlobal(nil)")
	}
}
