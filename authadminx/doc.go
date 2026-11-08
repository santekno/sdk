// Package authadminx is a minimal client for the Supabase Auth (GoTrue) admin
// API: invite, list, read, update, ban and delete users. It is authenticated
// with the project's service_role key, which grants unrestricted access, so it
// must only be used from trusted server-side code.
//
//	c, err := authadminx.New("https://abc.supabase.co", os.Getenv("SUPABASE_SERVICE_ROLE_KEY"))
//	u, err := c.Invite(ctx, "admin@example.com", authadminx.InviteOptions{
//		RedirectTo: "https://admin.example.com/set-password",
//		Data:       map[string]any{"name": "Admin"},
//	})
//	_, err = c.UpdateUser(ctx, u.ID, authadminx.UpdateUserParams{
//		AppMetadata: map[string]any{"role": "editor"},
//	})
//
// Safety properties: the key is never included in errors, String, GoString,
// or %v/%+v/%#v output; responses are limited to 1 MiB; redirects are not
// followed (a 3xx is reported as an *APIError); user ids are validated as
// UUIDs before they are placed in a URL path; and error messages are built
// from the HTTP status and a sanitised error code, never from the raw
// response body. The base URL must be https, except for loopback hosts used in
// local development.
//
// Revoking a user's sessions is not part of this client: GoTrue's public API
// only signs out the bearer of a user's own JWT, not an arbitrary user id.
package authadminx
