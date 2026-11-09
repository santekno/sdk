package authadminx

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
	"time"
)

// The request types must never be able to carry credentials, contact data,
// top-level role, confirmation flags, redirects or link requests.
func TestRequestTypesHaveNoDangerousFields(t *testing.T) {
	banned := map[string]bool{
		"password": true, "phone": true, "email_confirm": true, "phone_confirm": true,
		"role": true, "redirect_to": true, "type": true, "user_metadata": true,
		"should_soft_delete": true, "id": true, "aud": true, "encrypted_password": true,
	}
	for _, v := range []any{inviteRequest{}, inviteData{}, appMetadataRequest{}, banRequest{}, InviteOptions{}} {
		rt := reflect.TypeOf(v)
		for i := 0; i < rt.NumField(); i++ {
			f := rt.Field(i)
			if f.Type.Kind() == reflect.Map && rt != reflect.TypeOf(appMetadataRequest{}) {
				t.Errorf("%s.%s is a free-form map", rt, f.Name)
			}
			if f.Type.Kind() == reflect.Interface {
				t.Errorf("%s.%s is an interface", rt, f.Name)
			}
			tag := f.Tag.Get("json")
			for k := range banned {
				if tag == k || tag == k+",omitempty" {
					t.Errorf("%s has banned field %q", rt, k)
				}
			}
		}
	}
	if got := reflect.TypeOf(appMetadataRequest{}).Field(0).Type; got != reflect.TypeOf(map[string]string{}) {
		t.Errorf("app_metadata must be map[string]string, got %v", got)
	}
}

func TestUserDoesNotExposeExtras(t *testing.T) {
	rt := reflect.TypeOf(User{})
	for i := 0; i < rt.NumField(); i++ {
		switch rt.Field(i).Name {
		case "Identities", "Phone", "Factors", "EncryptedPassword":
			t.Errorf("User exposes %s", rt.Field(i).Name)
		}
	}
	var u User
	if err := json.Unmarshal([]byte(`{"id":"x","phone":"123","identities":[{}]}`), &u); err != nil || u.ID != "x" {
		t.Fatal(err)
	}
}

func TestTimeouts(t *testing.T) {
	if DefaultTimeout != 10*time.Second {
		t.Fatalf("DefaultTimeout = %v, want 10s (LAJ p2-03b D5)", DefaultTimeout)
	}
	custom := &http.Client{Timeout: time.Minute}
	tests := []struct {
		name string
		opts []Option
		want time.Duration
	}{
		{"default", nil, DefaultTimeout},
		{"with timeout", []Option{WithTimeout(3 * time.Second)}, 3 * time.Second},
		{"non-positive keeps default", []Option{WithTimeout(0), WithTimeout(-time.Second)}, DefaultTimeout},
		{"custom client timeout replaced", []Option{WithHTTPClient(custom)}, DefaultTimeout},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, err := New("https://abc.supabase.co", "k", tc.opts...)
			if err != nil {
				t.Fatal(err)
			}
			if c.hc.Timeout != tc.want {
				t.Errorf("timeout = %v, want %v", c.hc.Timeout, tc.want)
			}
			if c.hc.CheckRedirect == nil {
				t.Error("redirect policy not set")
			}
		})
	}
	if custom.Timeout != time.Minute || custom.CheckRedirect != nil {
		t.Error("injected client mutated")
	}
}
