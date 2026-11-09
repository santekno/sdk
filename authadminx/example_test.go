package authadminx_test

import (
	"context"
	"fmt"

	"github.com/santekno/sdk/authadminx"
)

func ExampleClient_SetAppMetadataField() {
	c, err := authadminx.New("https://abc.supabase.co", "service-role-key")
	if err != nil {
		return
	}
	fmt.Println(c)
	_ = func() {
		u, _ := c.Invite(context.Background(), "admin@example.com", authadminx.InviteOptions{Name: "Admin"})
		_, _ = c.SetAppMetadataField(context.Background(), u.ID, "role", "editor")
	}
	// Output: authadminx.Client{base: https://abc.supabase.co/auth/v1, key: [REDACTED]}
}
