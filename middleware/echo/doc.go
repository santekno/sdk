// Package echo provides Echo v4 framework middleware adapters for the Santekno
// Go SDK. It mirrors the existing middleware/gin package API surface 1:1 so
// services can choose either framework without re-learning the middleware
// conventions.
//
// # Available middleware
//
//   - [Logger]   — structured request log via logx
//   - [Recovery] — panic recovery returning HTTP 500
//   - [Tracing]  — X-Request-ID / traceparent propagation via ctxx
//   - [JWTAuth]  — Bearer-token validation via jwtx
//
// # Usage
//
//	import (
//	    "github.com/labstack/echo/v4"
//	    sdkecho "github.com/santekno/sdk/middleware/echo"
//	    "github.com/santekno/sdk/logx"
//	)
//
//	e := echo.New()
//	log := logx.New(logx.WithFormat("json"))
//	e.Use(sdkecho.Tracing())
//	e.Use(sdkecho.Recovery(log))
//	e.Use(sdkecho.Logger(log))
//
// # Why mirror middleware/gin
//
// Each Santekno service picks its own HTTP framework. The Phase 1 SDK shipped
// middleware/gin; Phase 3 of tools.santekno.com is built on Echo per its
// original PRD, hence this package. Both packages share types from
// [github.com/santekno/sdk/ctxx], [github.com/santekno/sdk/logx], and
// [github.com/santekno/sdk/jwtx], so context values, log fields, and JWT
// claims interoperate between services running different frameworks.
package echo
