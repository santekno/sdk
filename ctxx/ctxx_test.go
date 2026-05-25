package ctxx_test

import (
	"context"
	"testing"

	"github.com/santekno/sdk/ctxx"
)

func TestUserID(t *testing.T) {
	ctx := ctxx.WithUserID(context.Background(), "u-42")
	if got := ctxx.UserID(ctx); got != "u-42" {
		t.Errorf("UserID = %q, want u-42", got)
	}
	if got := ctxx.UserID(context.Background()); got != "" {
		t.Errorf("UserID(empty) = %q, want empty", got)
	}
	if got := ctxx.UserID(nil); got != "" { //nolint:staticcheck
		t.Errorf("UserID(nil) = %q, want empty", got)
	}
}

func TestRequestID(t *testing.T) {
	ctx := ctxx.WithRequestID(context.Background(), "req-1")
	if got := ctxx.RequestID(ctx); got != "req-1" {
		t.Errorf("RequestID = %q", got)
	}
	if got := ctxx.RequestID(nil); got != "" { //nolint:staticcheck
		t.Errorf("RequestID(nil) = %q", got)
	}
}

func TestTenantID(t *testing.T) {
	ctx := ctxx.WithTenantID(context.Background(), "t-9")
	if got := ctxx.TenantID(ctx); got != "t-9" {
		t.Errorf("TenantID = %q", got)
	}
	if got := ctxx.TenantID(nil); got != "" { //nolint:staticcheck
		t.Errorf("TenantID(nil) = %q", got)
	}
}
