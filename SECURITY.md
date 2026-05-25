# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| 0.x (dev) | ✅ |

## Reporting a Vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

Please report security vulnerabilities using one of these private channels:

1. **GitHub Security Advisories** (preferred): Navigate to the
   [Security tab](https://github.com/santekno/sdk/security/advisories/new)
   and click "Report a vulnerability".

2. **Email**: Send details to **security@santekno.com** with subject line
   `[SECURITY] <brief description>`.

Please include:
- Description of the vulnerability
- Steps to reproduce
- Affected versions
- Potential impact
- Suggested fix (if any)

## Response Timeline

| Stage | Target |
|-------|--------|
| Acknowledgement | 48 hours |
| Initial assessment | 5 business days |
| Fix + CVE assignment | 30 days (critical), 90 days (others) |
| Public disclosure | After fix is released |

## Security Principles

- No CGo anywhere in the SDK — reduces attack surface
- `cryptox` and `hashx` use only stdlib `crypto/` and `golang.org/x/crypto`
- Argon2id parameters meet or exceed OWASP 2026 recommendations
- Constant-time comparison used everywhere passwords or secrets are compared
- No reflection in hot paths
- All security packages (`cryptox`, `hashx`, `apikey`, `jwtx`) require ≥ 95% test coverage

## Acknowledgements

We thank all responsible disclosers. Credited researchers will be listed in
release notes unless they prefer to remain anonymous.
