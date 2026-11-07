# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- **Go toolchain requirement raised to 1.26.10** (from 1.25.10) in every module's `go.mod`, to clear the standard-library vulnerabilities `govulncheck` reports on Go 1.25.10. Consumers must build with Go 1.26.10 or newer: a module that depends on this SDK has its own `go` directive raised to at least 1.26.10 by `go get`, and builds with `GOTOOLCHAIN=local` on an older toolchain fail. CI, release, and vulnerability-scan workflows now use Go 1.26.10

### Fixed

- `healthx`: responses now carry `Content-Type: application/json`; the header was set after `WriteHeader` and dropped

### Added

- `timex.ParseISODuration` and `ErrInvalidISODuration`: parse the ISO 8601 `PnDTnHnMnS` subset (e.g. YouTube `contentDetails.duration`) into a `time.Duration`; rejects empty input, lowercase, years/months/weeks, signs, `,` fractions, fractions other than on seconds, `P`/`PT` without components, and `time.Duration` overflow
- `timex`: Gregorian to Hijri conversion with the Umm al-Qura calendar (`NewUmmAlQura`, `WithHijriMonthOffsets`, `FromCivilDate`, `MonthLength`, `ErrHijriOutOfRange`), covering 1356-1500 H with the month table from ICU; offsets move a month's start by one day and are rejected if any month would stop being 29 or 30 days long
- `middleware/chi`: net/http middleware for chi routers (RequestID, Logger, Recovery, Tracing, JWTAuth with ClaimsFromContext) and a pluggable JSON ErrorWriter; part of the root module, so it is versioned with every sdk tag
- `uuidx.NewV7`: time-ordered UUID version 7 (RFC 9562)
- `healthx.Handler.HideErrors`: omit check error text from public probe responses
- `httpx`: Resilient HTTP client with retry, circuit breaker, configurable timeouts
- `logx`: Structured logger interface backed by `log/slog`
- `tracex`: OTel-compatible tracer interface with no-op implementation
- `metricx`: Prometheus-compatible metrics interfaces
- `healthx`: Liveness and readiness HTTP health check handlers
- `timex`: Indonesian date/time formatting, business calendar, holiday registry (2024–2027)
- `validx`: NIK and NPWP parsing, IDR currency formatting, struct validation tags
- `jwtx`: JWT sign/verify (HS256, HS512, RS256) using stdlib crypto only
- `hashx`: Argon2id and bcrypt password hashing, HMAC helpers
- `cryptox`: AES-GCM, ChaCha20-Poly1305, RSA-OAEP, Ed25519 primitives
- `ctxx`: Typed context key helpers (user_id, request_id, tenant_id)
- `errx`: Structured application errors with codes, HTTP status, and detail chains
- `slicex`: Generic slice utilities (Map, Filter, Reduce, GroupBy, ParallelMap, …)
- `mapx`: Generic map utilities (Keys, Values, Entries, Merge, …)
- `stringx`: String helpers (ToCamel, ToSnake, Slugify, Truncate, …)
- `numx`: Generic numeric helpers (Min, Max, Clamp, Round, …)
- `uuidx`: UUID v4/v7 and ULID generation
- `ptrx`: Generic pointer helpers (Of, Deref)
- `convx`: Type conversion helpers
- `apikey`: API key generation and constant-time verification
- `middleware/gin`: JWT auth, logger, recovery, and tracing Gin middleware
- `internal/locale/id`: Indonesian month and weekday name maps
- `internal/version`: SDK version constant

## [v0.1.0-dev] — 2026-05-19

Initial development skeleton.

[Unreleased]: https://github.com/santekno/sdk/compare/v0.1.0-dev...HEAD
[v0.1.0-dev]: https://github.com/santekno/sdk/releases/tag/v0.1.0-dev
