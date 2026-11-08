package authadminx

import "time"

// User is an Auth user as returned by the admin API.
type User struct {
	ID               string         `json:"id"`
	Email            string         `json:"email"`
	Role             string         `json:"role"`
	AppMetadata      map[string]any `json:"app_metadata"`
	UserMetadata     map[string]any `json:"user_metadata"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	InvitedAt        *time.Time     `json:"invited_at"`
	ConfirmedAt      *time.Time     `json:"confirmed_at"`
	EmailConfirmedAt *time.Time     `json:"email_confirmed_at"`
	LastSignInAt     *time.Time     `json:"last_sign_in_at"`
	BannedUntil      *time.Time     `json:"banned_until"`
}

// InviteOptions configures Invite. Only a display name can be supplied; it is
// stored as user_metadata.name. There is intentionally no redirect URL and no
// free-form metadata map.
type InviteOptions struct {
	Name string // optional, at most 100 bytes, no control characters
}

// Wire types. They are deliberately minimal: none can carry password, phone,
// a top-level role, email_confirm, a redirect URL or a link request.
type inviteRequest struct {
	Email string      `json:"email"`
	Data  *inviteData `json:"data,omitempty"`
}

type inviteData struct {
	Name string `json:"name"`
}

type appMetadataRequest struct {
	AppMetadata map[string]string `json:"app_metadata"`
}

type banRequest struct {
	BanDuration string `json:"ban_duration"`
}

// ListResult is one page of users. Total is the server-reported number of
// users across all pages, or -1 when the server did not report it.
type ListResult struct {
	Users []User
	Total int
}
