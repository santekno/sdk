package authadminx_test

import (
	"context"
	"fmt"

	"github.com/santekno/sdk/authadminx"
)

func ExampleClient_Invite() {
	c, err := authadminx.New("https://abc.supabase.co", "service-role-key")
	if err != nil {
		return
	}
	fmt.Println(c)
	_ = func() {
		u, _ := c.Invite(context.Background(), "admin@example.com", authadminx.InviteOptions{RedirectTo: "https://admin.example.com/set-password"})
		_, _ = c.UpdateUser(context.Background(), u.ID, authadminx.UpdateUserParams{AppMetadata: map[string]any{"role": "editor"}})
	}
	// Output: authadminx.Client{base: https://abc.supabase.co/auth/v1, key: [REDACTED]}
}
