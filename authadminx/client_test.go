package authadminx_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/santekno/sdk/authadminx"
)

const (
	key = "super-secret-service-role-key"
	uid = "0b9c7d3e-1a2b-4c5d-8e9f-0123456789ab"
)

func newClient(t *testing.T, h http.HandlerFunc, opts ...authadminx.Option) *authadminx.Client {
	t.Helper()
	s := httptest.NewServer(h)
	t.Cleanup(s.Close)
	c, err := authadminx.New(s.URL, key, opts...)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func js(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}

func TestNew(t *testing.T) {
	tests := []struct {
		name, url, key string
		ok             bool
	}{
		{"https", "https://abc.supabase.co", key, true},
		{"https path slash", "https://abc.supabase.co/", key, true},
		{"loopback http", "http://127.0.0.1:54321", key, true},
		{"localhost http", "http://localhost:54321", key, true},
		{"ipv6 loopback", "http://[::1]:1", key, true},
		{"remote http", "http://abc.supabase.co", key, false},
		{"bad scheme", "ftp://abc.supabase.co", key, false},
		{"no host", "https://", key, false},
		{"userinfo", "https://u:p@abc.supabase.co", key, false},
		{"query", "https://abc.supabase.co?x=1", key, false},
		{"unparsable", "://x", key, false},
		{"empty key", "https://abc.supabase.co", "", false},
		{"key whitespace", "https://abc.supabase.co", "a b", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := authadminx.New(tc.url, tc.key)
			if (err == nil) != tc.ok {
				t.Fatalf("err=%v ok=%v", err, tc.ok)
			}
			if err != nil && !errors.Is(err, authadminx.ErrInvalidArgument) {
				t.Fatalf("want ErrInvalidArgument: %v", err)
			}
			if err != nil && tc.key != "" && strings.Contains(err.Error(), tc.key) {
				t.Fatal("key leaked")
			}
		})
	}
}

func TestOptions(t *testing.T) {
	custom := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":"x"}`)), Header: http.Header{}}, nil
	})}
	c, err := authadminx.New("https://abc.supabase.co", key,
		authadminx.WithHTTPClient(custom), authadminx.WithHTTPClient(nil), authadminx.WithTimeout(0), authadminx.WithTimeout(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	u, err := c.GetUser(context.Background(), uid)
	if err != nil || u.ID != "x" {
		t.Fatalf("%v %v", u, err)
	}
	if custom.Timeout != 0 {
		t.Fatal("injected client must not be mutated")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRequests(t *testing.T) {
	user := `{"id":"` + uid + `","email":"a@b.co","app_metadata":{"role":"editor"},"banned_until":"2030-01-01T00:00:00Z"}`
	tests := []struct {
		name       string
		call       func(*authadminx.Client) error
		method     string
		path, rawQ string
		body       string
		reply      string
	}{
		{"invite", func(c *authadminx.Client) error {
			u, err := c.Invite(context.Background(), "a@b.co", authadminx.InviteOptions{Name: "N"})
			if err == nil && u.ID != uid {
				err = errors.New("bad user")
			}
			return err
		}, "POST", "/auth/v1/invite", "", `{"email":"a@b.co","data":{"name":"N"}}`, user},
		{"invite plain", func(c *authadminx.Client) error {
			_, err := c.Invite(context.Background(), "a@b.co", authadminx.InviteOptions{})
			return err
		}, "POST", "/auth/v1/invite", "", `{"email":"a@b.co"}`, user},
		{"get", func(c *authadminx.Client) error { _, err := c.GetUser(context.Background(), uid); return err },
			"GET", "/auth/v1/admin/users/" + uid, "", "", user},
		{"set field", func(c *authadminx.Client) error {
			u, err := c.SetAppMetadataField(context.Background(), uid, "role", "editor")
			if err == nil && (u.BannedUntil == nil || u.AppMetadata["role"] != "editor") {
				err = errors.New("bad decode")
			}
			return err
		}, "PUT", "/auth/v1/admin/users/" + uid, "", `{"app_metadata":{"role":"editor"}}`, user},
		{"ban", func(c *authadminx.Client) error { _, err := c.SetBanned(context.Background(), uid, true); return err },
			"PUT", "/auth/v1/admin/users/" + uid, "", `{"ban_duration":"876000h"}`, user},
		{"unban", func(c *authadminx.Client) error { _, err := c.SetBanned(context.Background(), uid, false); return err },
			"PUT", "/auth/v1/admin/users/" + uid, "", `{"ban_duration":"none"}`, user},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := newClient(t, func(w http.ResponseWriter, r *http.Request) {
				b, _ := io.ReadAll(r.Body)
				if r.Method != tc.method || r.URL.Path != tc.path || r.URL.RawQuery != tc.rawQ {
					t.Errorf("got %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
				}
				if tc.body != "" && string(b) != tc.body {
					t.Errorf("body %s", b)
				}
				if r.Header.Get("Authorization") != "Bearer "+key || r.Header.Get("apikey") != key {
					t.Error("auth headers")
				}
				if !strings.HasPrefix(r.Header.Get("User-Agent"), "santekno-sdk-authadminx") {
					t.Error("user agent")
				}
				js(w, 200, tc.reply)
			})
			if err := tc.call(c); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestListUsers(t *testing.T) {
	tests := []struct {
		name      string
		total     string
		wantTotal int
	}{{"with total", "42", 42}, {"missing", "", -1}, {"garbage", "x", -1}, {"negative", "-3", -1}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := newClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/auth/v1/admin/users" || r.URL.RawQuery != "page=2&per_page=50" {
					t.Errorf("url %s", r.URL)
				}
				if tc.total != "" {
					w.Header().Set("X-Total-Count", tc.total)
				}
				js(w, 200, `{"users":[{"id":"`+uid+`"},{"id":"b"}],"aud":"authenticated"}`)
			})
			res, err := c.ListUsers(context.Background(), 2, 50)
			if err != nil || len(res.Users) != 2 || res.Total != tc.wantTotal {
				t.Fatalf("%+v %v", res, err)
			}
		})
	}
}

func TestInvalidArguments(t *testing.T) {
	c, _ := authadminx.New("https://abc.supabase.co", key)
	ctx := context.Background()
	long := strings.Repeat("a", 250) + "@b.co"
	tests := []struct {
		name string
		call func() error
	}{
		{"id not uuid", func() error { _, e := c.GetUser(ctx, "../admin"); return e }},
		{"id set field", func() error { _, e := c.SetAppMetadataField(ctx, "1", "role", "editor"); return e }},
		{"ban id", func() error { _, e := c.SetBanned(ctx, "", true); return e }},
		{"key uppercase", func() error { _, e := c.SetAppMetadataField(ctx, uid, "Role", "x"); return e }},
		{"key empty", func() error { _, e := c.SetAppMetadataField(ctx, uid, "", "x"); return e }},
		{"key dot", func() error { _, e := c.SetAppMetadataField(ctx, uid, "a.b", "x"); return e }},
		{"key long", func() error { _, e := c.SetAppMetadataField(ctx, uid, strings.Repeat("a", 33), "x"); return e }},
		{"value empty", func() error { _, e := c.SetAppMetadataField(ctx, uid, "role", ""); return e }},
		{"value space", func() error { _, e := c.SetAppMetadataField(ctx, uid, "role", "a b"); return e }},
		{"value upper", func() error { _, e := c.SetAppMetadataField(ctx, uid, "role", "Owner"); return e }},
		{"value long", func() error { _, e := c.SetAppMetadataField(ctx, uid, "role", strings.Repeat("a", 65)); return e }},
		{"name control", func() error { _, e := c.Invite(ctx, "a@b.co", authadminx.InviteOptions{Name: "a\nb"}); return e }},
		{"name long", func() error {
			_, e := c.Invite(ctx, "a@b.co", authadminx.InviteOptions{Name: strings.Repeat("a", 101)})
			return e
		}},
		{"name utf8", func() error { _, e := c.Invite(ctx, "a@b.co", authadminx.InviteOptions{Name: "\xff"}); return e }},
		{"email empty", func() error { _, e := c.Invite(ctx, "", authadminx.InviteOptions{}); return e }},
		{"email crlf", func() error { _, e := c.Invite(ctx, "a@b.co\r\nBcc: x@y.z", authadminx.InviteOptions{}); return e }},
		{"email display name", func() error { _, e := c.Invite(ctx, "A <a@b.co>", authadminx.InviteOptions{}); return e }},
		{"email long", func() error { _, e := c.Invite(ctx, long, authadminx.InviteOptions{}); return e }},
		{"page 0", func() error { _, e := c.ListUsers(ctx, 0, 10); return e }},
		{"perPage 0", func() error { _, e := c.ListUsers(ctx, 1, 0); return e }},
		{"perPage big", func() error { _, e := c.ListUsers(ctx, 1, 1001); return e }},
		{"nil ctx", func() error { _, e := c.GetUser(nil, uid); return e }}, //nolint:staticcheck
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.call(); !errors.Is(err, authadminx.ErrInvalidArgument) {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestAPIErrors(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		sentinel error
		code     string
	}{
		{"404", 404, `{"code":404,"error_code":"user_not_found","msg":"User not found"}`, authadminx.ErrNotFound, "user_not_found"},
		{"401", 401, `{}`, authadminx.ErrUnauthorized, ""},
		{"403", 403, `{"error_code":"not_admin"}`, authadminx.ErrUnauthorized, "not_admin"},
		{"409", 409, `{"error_code":"conflict"}`, authadminx.ErrConflict, "conflict"},
		{"422", 422, `{"error_code":"email_exists","msg":"a@b.co already registered " }`, authadminx.ErrUnprocessable, "email_exists"},
		{"429", 429, `{"error_code":"over_email_send_rate_limit"}`, authadminx.ErrRateLimited, "over_email_send_rate_limit"},
		{"500 html", 500, `<html>secret-body</html>`, nil, ""},
		{"unsafe code", 400, `{"error_code":"bad code\nwith newline and secret-body"}`, nil, ""},
		{"302", 302, ``, nil, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := newClient(t, func(w http.ResponseWriter, r *http.Request) {
				if tc.status == 302 {
					w.Header().Set("Location", "http://evil.example/")
				}
				js(w, tc.status, tc.body)
			})
			_, err := c.GetUser(context.Background(), uid)
			var ae *authadminx.APIError
			if !errors.As(err, &ae) || ae.Status != tc.status || ae.Code != tc.code {
				t.Fatalf("got %#v", err)
			}
			if tc.sentinel != nil && !errors.Is(err, tc.sentinel) {
				t.Fatalf("want sentinel %v", tc.sentinel)
			}
			for _, other := range []error{authadminx.ErrNotFound, authadminx.ErrConflict, authadminx.ErrUnprocessable, authadminx.ErrRateLimited, authadminx.ErrUnauthorized} {
				if other != tc.sentinel && errors.Is(err, other) {
					t.Fatalf("matches %v", other)
				}
			}
			for _, leak := range []string{"secret-body", "already registered", "User not found", key} {
				if strings.Contains(err.Error(), leak) {
					t.Fatalf("error leaks %q: %v", leak, err)
				}
			}
			if !strings.Contains(err.Error(), fmt.Sprint(tc.status)) {
				t.Fatal("status missing")
			}
		})
	}
}

func TestRedirectNotFollowed(t *testing.T) {
	calls := []struct {
		name string
		do   func(*authadminx.Client) error
	}{
		{"get", func(c *authadminx.Client) error { _, e := c.GetUser(context.Background(), uid); return e }},
		{"invite", func(c *authadminx.Client) error {
			_, e := c.Invite(context.Background(), "a@b.co", authadminx.InviteOptions{})
			return e
		}},
		{"set field", func(c *authadminx.Client) error {
			_, e := c.SetAppMetadataField(context.Background(), uid, "role", "editor")
			return e
		}},
		{"ban", func(c *authadminx.Client) error { _, e := c.SetBanned(context.Background(), uid, true); return e }},
	}
	for _, tc := range calls {
		t.Run(tc.name, func(t *testing.T) {
			var hits atomic.Int32
			var leaked atomic.Bool
			evil := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits.Add(1)
				if r.Header.Get("apikey") != "" || r.Header.Get("Authorization") != "" {
					leaked.Store(true)
				}
			}))
			defer evil.Close()
			var first atomic.Int32
			c := newClient(t, func(w http.ResponseWriter, r *http.Request) {
				first.Add(1)
				http.Redirect(w, r, evil.URL, http.StatusTemporaryRedirect)
			})
			err := tc.do(c)
			var ae *authadminx.APIError
			if !errors.As(err, &ae) || ae.Status != 307 || hits.Load() != 0 || leaked.Load() || first.Load() != 1 {
				t.Fatalf("err=%v hits=%d first=%d", err, hits.Load(), first.Load())
			}
		})
	}
}

func TestResponseLimit(t *testing.T) {
	tests := []struct {
		name string
		size int
		ok   bool
	}{{"at limit", authadminx.MaxResponseBytes, true}, {"over limit", authadminx.MaxResponseBytes + 1, false}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := newClient(t, func(w http.ResponseWriter, r *http.Request) {
				pad := tc.size - len(`{"id":"x","email":""}`)
				js(w, 200, `{"id":"x","email":"`+strings.Repeat("a", pad)+`"}`)
			})
			_, err := c.GetUser(context.Background(), uid)
			if tc.ok && err != nil {
				t.Fatal(err)
			}
			if !tc.ok && !errors.Is(err, authadminx.ErrResponseTooLarge) {
				t.Fatalf("got %v", err)
			}
		})
	}
	t.Run("error body over limit", func(t *testing.T) {
		c := newClient(t, func(w http.ResponseWriter, r *http.Request) {
			js(w, 500, strings.Repeat("x", authadminx.MaxResponseBytes+10))
		})
		if _, err := c.GetUser(context.Background(), uid); !errors.Is(err, authadminx.ErrResponseTooLarge) {
			t.Fatalf("got %v", err)
		}
	})
}

func TestInvalidResponse(t *testing.T) {
	c := newClient(t, func(w http.ResponseWriter, r *http.Request) { js(w, 200, `not json`) })
	if _, err := c.GetUser(context.Background(), uid); !errors.Is(err, authadminx.ErrInvalidResponse) {
		t.Fatalf("got %v", err)
	}
}

func TestTimeoutAndContext(t *testing.T) {
	release := make(chan struct{})
	c := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}, authadminx.WithTimeout(50*time.Millisecond))
	defer close(release)
	_, err := c.GetUser(context.Background(), uid)
	if err == nil || strings.Contains(err.Error(), key) || strings.Contains(err.Error(), "127.0.0.1") {
		t.Fatalf("got %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.GetUser(ctx, uid); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}

func TestBodyReadError(t *testing.T) {
	hc := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(errReader{})}, nil
	})}
	c, _ := authadminx.New("https://abc.supabase.co", key, authadminx.WithHTTPClient(hc))
	if _, err := c.GetUser(context.Background(), uid); err == nil {
		t.Fatal("want error")
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func TestKeyNeverLeaks(t *testing.T) {
	c, _ := authadminx.New("https://abc.supabase.co", key)
	var buf bytes.Buffer
	lg := log.New(&buf, "", 0)
	lg.Printf("%v %+v %#v %s %q %d", c, c, c, c, c, c)
	lg.Println(c, *c)
	lg.Printf("%v %+v %#v", *c, *c, *c)
	out := fmt.Sprint(c, c.String(), c.GoString())
	b, _ := json.Marshal(c)
	if strings.Contains(buf.String()+out+string(b), key) {
		t.Fatalf("key leaked: %s", buf.String())
	}
	if !strings.Contains(c.String(), "REDACTED") {
		t.Fatal("no redaction marker")
	}
}

func TestKeyNotInTransportError(t *testing.T) {
	hc := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("dial failed for %s auth=%s", r.URL, r.Header.Get("Authorization")[:0])
	})}
	c, _ := authadminx.New("https://abc.supabase.co", key, authadminx.WithHTTPClient(hc))
	_, err := c.GetUser(context.Background(), uid)
	if err == nil || strings.Contains(err.Error(), key) {
		t.Fatalf("got %v", err)
	}
}
