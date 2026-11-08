// Package authadminx is a deliberately narrow client for the Supabase Auth
// (GoTrue) admin API: invite a user, list, read, set one app_metadata string
// field, and ban or unban. It is authenticated with the project's service_role
// key, which grants unrestricted access, so the exposed surface is kept to the
// minimum a holder of that key needs; it must only be used from trusted
// server-side code.
//
//	c, err := authadminx.New("https://abc.supabase.co", os.Getenv("SUPABASE_SERVICE_ROLE_KEY"))
//	u, err := c.Invite(ctx, "admin@example.com", authadminx.InviteOptions{Name: "Admin"})
//	_, err = c.SetAppMetadataField(ctx, u.ID, "role", "editor")
//	_, err = c.SetBanned(ctx, u.ID, true)
//
// Not provided on purpose: deleting users, generate_link, creating users with
// a password, redirect_to on invites, free-form metadata maps, and writing the
// top-level Postgres role, email, phone or confirmation fields.
//
// Safety properties: the key is never included in errors, String, GoString,
// or %v/%+v/%#v output; the key is only ever sent to the configured base URL
// (redirects are not followed, a 3xx is reported as an *APIError); responses
// are limited to 1 MiB; user ids are validated as UUIDs before they are placed
// in a URL path; and error messages are built from the HTTP status and a
// sanitised error code, never from the raw response body. The base URL must be
// https, except for loopback hosts used in local development.
//
// A ban does not invalidate access tokens that were already issued. Revoking a
// user's sessions is not part of this client; callers must do it themselves
// when needed.
package authadminx
