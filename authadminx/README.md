# authadminx

Narrow Supabase Auth (GoTrue) admin client for code that holds the
`service_role` key.

API: `New`, `Invite(email, InviteOptions{Name})`, `ListUsers`, `GetUser`,
`SetAppMetadataField(id, key, value)`, `SetBanned(id, bool)`.

Intentionally absent: `DeleteUser`, `generate_link`, create-with-password,
`redirect_to`, free-form metadata maps, and any write to the top-level
`role`, `email`, `phone`, `password` or `email_confirm`.

## Limits and guarantees

- HTTPS required (loopback excepted); redirects are never followed, so the
  `apikey` and `Authorization` headers cannot reach another host.
- Explicit timeout (10 s default, covers connect, headers and body) and
  transport; responses capped at 1 MiB.
- User ids must be UUIDs; metadata key `^[a-z_]{1,32}$`, value `^[a-z0-9_-]{1,64}$`.
- The key is redacted under every `fmt` verb; errors never contain response
  bodies, request URLs or the key.

## Errors

| Error | When |
| --- | --- |
| `*APIError` (`Status`, `Code`) | Non-2xx answer; `Code` is GoTrue's `error_code` (body or `X-Sb-Error-Code`) when it matches `^[a-z0-9_]{1,64}$` |
| `ErrEmailExists` | `Code` is `email_exists` or `user_already_exists` (Invite of a confirmed user: 422) |
| `ErrNotFound`, `ErrConflict`, `ErrUnprocessable`, `ErrRateLimited`, `ErrUnauthorized` | Status 404, 409, 422, 429, 401/403 |
| `ErrTransport` | Dial, DNS, TLS, timeout, cancellation or body read failure; `context.Canceled`, `context.DeadlineExceeded` and `net.Error` stay matchable |
| `ErrInvalidArgument` | Input rejected before any request |
| `ErrResponseTooLarge`, `ErrInvalidResponse` | Body over 1 MiB, undecodable success body |

`Invite` for an email whose user has not confirmed yet sends a new invitation
(re-invite); a confirmed user is rejected with `ErrEmailExists`.

## Bans and sessions

`SetBanned(id, true)` stops new sign-ins and token refresh, but **does not
invalidate access tokens already issued**; they remain valid until they expire.
Callers must revoke sessions themselves when immediate cut-off is required.
This client does not provide session revocation.
