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

// InviteOptions configures Invite. Both fields are optional.
type InviteOptions struct {
	RedirectTo string         // absolute http(s) URL the invite link lands on
	Data       map[string]any // stored as user_metadata
}

// UpdateUserParams selects the fields UpdateUser changes. Empty fields are
// left untouched. Note that AppMetadata is merged by the server at top level.
type UpdateUserParams struct {
	AppMetadata  map[string]any `json:"app_metadata,omitempty"`
	UserMetadata map[string]any `json:"user_metadata,omitempty"`
	// BanDuration is a Go-style duration such as "720h", or "none" to lift a ban.
	BanDuration string `json:"ban_duration,omitempty"`
}

// ListResult is one page of users. Total is the server-reported number of
// users across all pages, or -1 when the server did not report it.
type ListResult struct {
	Users []User
	Total int
}
