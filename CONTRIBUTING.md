# Contributing to Santekno Go SDK

Thank you for your interest in contributing! This guide covers the full workflow.

## Prerequisites

- Go ≥ 1.25
- `golangci-lint` v2 (`go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`)
- `make`

## Fork & Branch Workflow

1. Fork the repository on GitHub.
2. Clone your fork: `git clone https://github.com/<you>/sdk.git`
3. Create a feature branch: `git checkout -b feat/my-feature`
4. Make your changes, add tests, ensure coverage thresholds are met.
5. Push and open a pull request against `main`.

## Running Tests

```bash
make test       # run all tests
make lint       # run golangci-lint v2
make bench      # run benchmarks
make coverage   # show coverage report
```

## Coverage Requirements

| Package category | Minimum coverage |
|------------------|-----------------|
| Core utilities (`slicex`, `mapx`, `stringx`, `numx`, `ptrx`, `convx`) | ≥ 85% |
| Infrastructure wrappers (`httpx`, `logx`, `timex`, `validx`) | ≥ 85% |
| Security packages (`cryptox`, `hashx`, `apikey`, `jwtx`) | ≥ 95% |

## Conventional Commits

All commit messages must follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(httpx): add WithMaxBodySize option
fix(timex): correct IsIndonesianHoliday for 2025 Lebur dates
docs(validx): add ExampleParseNIK godoc
test(hashx): add fuzz corpus for VerifyPassword
chore(deps): bump golang.org/x/crypto to v0.52
```

Types: `feat`, `fix`, `docs`, `test`, `chore`, `refactor`, `perf`, `ci`, `build`

Breaking changes: add `!` suffix or `BREAKING CHANGE:` footer.

## PR Review SLA

Maintainers aim to review PRs within 72 hours of submission.

## CODEOWNERS

The following packages require 2 reviewers (Lead Maintainer + 1 additional):

- `cryptox/`
- `hashx/`
- `apikey/`
- `jwtx/`

All other packages require the Lead Maintainer review.

## Code Style

- `gofmt` / `goimports` formatting is enforced by CI.
- No CGo anywhere in the SDK.
- No `init()` functions with side effects.
- No global mutable state in library code.
- All public APIs must have godoc comments.
