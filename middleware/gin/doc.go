// Package gin provides Gin framework middleware adapters for the Santekno Go SDK.
//
// Available middleware:
//   - JWTAuth: validates Bearer tokens using jwtx and injects claims into the context
//   - Logger: structured request/response logging via logx
//   - Recovery: panic recovery with structured error logging
//   - Tracing: OTel span creation and W3C TraceContext propagation
//
// Usage:
//
//	r := gin.New()
//	l := logx.New()
//	key := jwtx.HS256([]byte("secret"))
//	r.Use(ginmw.Logger(l), ginmw.Recovery(l), ginmw.Tracing())
//	r.GET("/me", ginmw.JWTAuth(key), myHandler)
package gin
