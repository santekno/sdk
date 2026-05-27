package ctxx

import "context"

type (
	userIDKey    struct{}
	requestIDKey struct{}
	tenantIDKey  struct{}
)

// WithUserID returns a new context with userID attached.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey{}, userID)
}

// UserID returns the user ID stored in ctx, or "" if none.
func UserID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(userIDKey{}).(string)
	return v
}

// WithRequestID returns a new context with requestID attached.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

// RequestID returns the request ID stored in ctx, or "" if none.
func RequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(requestIDKey{}).(string)
	return v
}

// WithTenantID returns a new context with tenantID attached.
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantIDKey{}, tenantID)
}

// TenantID returns the tenant ID stored in ctx, or "" if none.
func TenantID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(tenantIDKey{}).(string)
	return v
}
