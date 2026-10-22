// Package chi provides net/http middleware for chi routers, built on the
// Santekno Go SDK.
//
// Every middleware has the standard func(http.Handler) http.Handler shape, so
// it plugs into chi's r.Use / r.With and also works with plain net/http.
// Unlike the gin and echo adapters, this package lives in the root sdk module:
// chi brings no transitive dependencies, so it ships with every sdk release.
//
// Available middleware:
//   - RequestID: propagates X-Request-ID (or generates one) into ctxx
//   - Logger: one structured logx line per request, with the chi route pattern
//   - Recovery: turns panics into a 500 JSON error and logs the stack
//   - Tracing: starts a tracex span per request named "METHOD /route/{pattern}"
//   - JWTAuth: verifies Bearer tokens with jwtx and stores the claims in the context
//
// Errors are written as JSON by an ErrorWriter. The default writes
//
//	{"error":{"code":"UNAUTHORIZED","message":"missing Authorization header"}}
//
// and WithErrorWriter lets an application keep one error shape across its API.
//
// Usage:
//
//	r := chi.NewRouter()
//	l := logx.New()
//	key := jwtx.HS256([]byte(os.Getenv("JWT_SECRET")))
//	r.Use(chimw.RequestID(), chimw.Logger(l), chimw.Recovery(l), chimw.Tracing(tracex.Global()))
//	r.With(chimw.JWTAuth(key)).Get("/me", meHandler)
package chi
