# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

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
