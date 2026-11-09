# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- `messagingx`: the default HTTP client timeout is 15 seconds instead of 65. `Poll` calls `getUpdates` with `timeout=0`, so the long-poll headroom was unused. Pass `WithHTTPClient` to choose another timeout
- **Go toolchain requirement raised to 1.26.10** (from 1.25.10) in every module's `go.mod`, to clear the standard-library vulnerabilities `govulncheck` reports on Go 1.25.10. Consumers must build with Go 1.26.10 or newer: a module that depends on this SDK has its own `go` directive raised to at least 1.26.10 by `go get`, and builds with `GOTOOLCHAIN=local` on an older toolchain fail. CI, release, and vulnerability-scan workflows now use Go 1.26.10

### Fixed

- `messagingx`: errors no longer leak the Telegram bot token. The Bot API puts the token in the request path, and transport errors were returned as `*url.Error` with the full URL. Request-building, transport, and decode errors are now rewrapped as `messagingx: telegram <method>: <stage>: <cause>` without the `*url.Error` layer; a cause whose text still contains the token (for example from a custom `RoundTripper`) is replaced by a redacted copy. `errors.Is` with `context.Canceled`/`context.DeadlineExceeded` and `errors.As` with `net.Error` (`Timeout`) keep working. Behaviour change: `errors.As(err, *url.Error)` no longer matches
- `messagingx`: an `ok: false` response reports `api error <code> <text>` (plus `retry after` when Telegram sends it). Telegram's free-text `description` is kept only for a fixed list of data-free messages (such as `Bad Request: chat not found`); any other description is replaced by the HTTP status text, because it can echo message content. A non-JSON error page is reported by its HTTP status
- `messagingx`: `WithHTTPClient(nil)` keeps the default client instead of causing a nil-pointer panic on the first call
- `ratelimitx` is now part of the root module instead of a nested module with no release tag, so it can be pinned with `go get github.com/santekno/sdk@vX.Y.Z` and imports as `github.com/santekno/sdk/ratelimitx` as before. The package has no external dependencies. Breaking only for anyone who required the never-tagged module `github.com/santekno/sdk/ratelimitx` at a pseudo-version: drop that `require` (and any `replace`) and depend on `github.com/santekno/sdk` instead, otherwise Go reports an ambiguous import. Added the `example_test.go` that `doc.go` already referenced

- `storagex.LocalStorage` keeps the key's directory prefix and writes atomically. `v/a.webp` is stored at `dir/v/a.webp` (parent directories are created with mode 0750) instead of `dir/a.webp`, so `v/a.webp` and `u/a.webp` no longer overwrite each other. `Put` writes to a temporary file in the destination directory, syncs it, and renames it over the key, so a failed or canceled copy leaves no partial object and keeps the previous one; files are created with mode 0600 (was 0666 minus umask) and `Put` now honors context cancellation. Keys that are empty, longer than 1024 bytes, not valid UTF-8, absolute, or contain `\`, a control character, an empty segment, a `.`/`..` segment, or a `.storagex-tmp-` segment are rejected by `Put`, `Get`, and `Delete` with the new `storagex.ErrInvalidKey` (the error does not echo the key), and all file operations go through `os.Root`, so a symbolic link cannot lead outside the directory. `Get` and `Delete` on a key that names a directory, or that lies below an existing file (`ENOTDIR`), return an error matching `fs.ErrNotExist`. Filesystem errors no longer echo the key or any path: the message is `storagex: local <op>: <system error>` and the error still unwraps to the `*fs.PathError`/`*os.LinkError` (`os.IsNotExist` no longer matches; use `errors.Is(err, fs.ErrNotExist)`). A process killed during `Put` can leave a `.storagex-tmp-*` file, which is never read as an object. Flat keys such as `report.csv` map to the same files as in v0.5.0
- `healthx`: responses now carry `Content-Type: application/json`; the header was set after `WriteHeader` and dropped
- `jwtx`: the `Verify` algorithm mismatch error no longer echoes the token's `alg` header

### Added

- `authadminx`: narrow Supabase Auth (GoTrue) admin client for service_role holders: `Invite` (email and optional `Name` only, no `redirect_to`, no free-form metadata; re-sends the invitation to an unconfirmed user), `ListUsers`, `GetUser`, `SetAppMetadataField` (key `^[a-z_]{1,32}$`, value `^[a-z0-9_-]{1,64}$`, never the top-level role, email, password or phone) and `SetBanned`. No user deletion, `generate_link` or create-with-password. HTTPS required except loopback, 10 second default timeout (`WithTimeout`), injectable `http.Client` (copied, never mutated), 1 MiB response limit, redirects not followed (credentials never leave the base host), user ids validated as UUIDs. Typed errors: `*APIError` (status and sanitised `error_code`, never the response body) with sentinels `ErrNotFound`, `ErrConflict`, `ErrUnprocessable`, `ErrRateLimited`, `ErrUnauthorized` and `ErrEmailExists`; `ErrTransport` for dial, DNS, timeout, cancellation and body read failures, without the request URL and with `context` and `net.Error` causes still matchable; the key is redacted from errors (including a custom transport echoing it) and every `fmt` verb. A ban does not invalidate issued access tokens; session revocation is not included
- `hashx.HMACSHA1`: HMAC-SHA-1 for protocol compatibility only (e.g. WebSub `X-Hub-Signature`); not for new designs. Verify with `ConstantTimeEqual`
- `jwtx`: ES256 verification (`ES256Verify`, JWS `r||s` signatures)
- `jwtx`: `ParseKeySet` builds a `StaticKeySet` from a JWKS document (EC P-256 and RSA 2048 to 8192 bits); other keys are skipped
- `jwtx`: `JWKSClient` fetches a JWKS over HTTPS (5 second timeout, 64 KiB limit, no redirects), caches it for 10 minutes or a shorter `max-age`, refreshes at most once per 30 seconds with a single in-flight fetch, and keeps the old keys when a refresh fails, for at most 24 hours after the last successful fetch (`WithMaxStale`) before returning `ErrKeySetUnavailable`
- `jwtx`: `VerifyWithKeySet` verifies third-party tokens against a `KeySet` with a required `kid`, a case-sensitive algorithm allow-list, an algorithm bound to the key, required `exp`, and `WithLeeway`/`WithContext` options
- `middleware/chi`: `ClientIP(trustedHeader)` resolves the client IP from exactly one configured trusted header (or `RemoteAddr` when unset, absent, or malformed) and `ClientIPFromContext` reads it. `X-Forwarded-For`, `True-Client-IP` and every other header are never read; multi-value, comma-separated, zoned and non-IP values are rejected, ports are stripped, and IPv4-mapped IPv6 is normalized, so a forged header cannot pick the caller's rate-limit key
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
