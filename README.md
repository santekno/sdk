# Santekno Go SDK

[![CI](https://github.com/santekno/sdk/actions/workflows/ci.yml/badge.svg)](https://github.com/santekno/sdk/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/go-%3E%3D1.25-blue)](https://go.dev)
[![Go Reference](https://pkg.go.dev/badge/github.com/santekno/sdk.svg)](https://pkg.go.dev/github.com/santekno/sdk)
[![codecov](https://codecov.io/gh/santekno/sdk/branch/main/graph/badge.svg)](https://codecov.io/gh/santekno/sdk)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/santekno/sdk)](https://goreportcard.com/report/github.com/santekno/sdk)

A MIT-licensed, generics-first Go utility SDK spanning 20+ sub-packages — HTTP resilience,
structured logging, JWT auth, validation, observability, and first-class Indonesian locale support.

## Install

```bash
go get github.com/santekno/sdk
```

Or install only what you need:

```bash
go get github.com/santekno/sdk/httpx
go get github.com/santekno/sdk/logx
go get github.com/santekno/sdk/timex
```

## Quickstart

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/santekno/sdk/httpx"
    "github.com/santekno/sdk/logx"
    "github.com/santekno/sdk/timex"
    "github.com/santekno/sdk/validx"
)

func main() {
    // Structured logger
    logger := logx.New(logx.WithLevel("info"))

    // Resilient HTTP client with retries + circuit breaker
    client := httpx.New(
        httpx.WithTimeout(5*time.Second),
        httpx.WithRetries(3),
        httpx.WithLogger(logger),
    )

    var result map[string]any
    if err := client.GetJSON(context.Background(), "https://api.example.com/health", &result); err != nil {
        log.Fatal(err)
    }

    // Indonesian date formatting
    now := time.Now().In(timex.Jakarta)
    logger.Info("date", "formatted", timex.FormatID(now, timex.LongID))

    // NIK parsing
    info, _ := validx.ParseNIK("3201010101800001")
    logger.Info("nik", "province", info.Province, "gender", info.Gender)
}
```

## Packages

| Package | Description |
|---------|-------------|
| `httpx` | Resilient HTTP client with retry, circuit breaker, timeouts |
| `logx` | Structured logger interface (slog-backed) |
| `timex` | Indonesian date/time formatting, business calendar |
| `validx` | NIK/NPWP/IDR validation and parsing |
| `jwtx` | JWT sign/verify (HS256, HS512, RS256) |
| `hashx` | Argon2id and bcrypt password hashing |
| `cryptox` | AES-GCM, ChaCha20, RSA-OAEP, Ed25519 encryption |
| `errx` | Structured application errors with codes |
| `ctxx` | Typed context key helpers (user_id, request_id, tenant_id) |
| `tracex` | Tracer interface + no-op implementation |
| `metricx` | Metrics interfaces (Counter, Gauge, Histogram) |
| `healthx` | Liveness and readiness HTTP health check handlers |
| `slicex` | Generic slice utilities (Map, Filter, Reduce, GroupBy, …) |
| `mapx` | Generic map utilities (Keys, Values, Merge, …) |
| `stringx` | String helpers (ToCamel, ToSnake, Slugify, …) |
| `numx` | Generic numeric helpers (Min, Max, Clamp, Round, …) |
| `uuidx` | UUID v4/v7 and ULID generation |
| `ptrx` | Generic pointer helpers (Of, Deref) |
| `convx` | Type conversion helpers |
| `apikey` | API key generation and verification |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). PRs welcome!

## License

MIT — see [LICENSE](LICENSE).
