// Package ctxx provides typed context key helpers for cross-package values
// such as user ID, request ID, and tenant ID.
//
//	ctx = ctxx.WithUserID(ctx, "u-42")
//	uid := ctxx.UserID(ctx)
package ctxx
