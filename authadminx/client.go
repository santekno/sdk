package authadminx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultTimeout is the per-request timeout of the default HTTP client.
	DefaultTimeout = 5 * time.Second
	// MaxResponseBytes is the largest response body accepted.
	MaxResponseBytes = 1 << 20
	// BanForever is the BanDuration used by Ban (about 100 years).
	BanForever = "876000h"
	// BanNone lifts a ban when used as BanDuration.
	BanNone = "none"

	maxPerPage = 1000
	userAgent  = "santekno-sdk-authadminx/1"
)

var (
	uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	codeRe = regexp.MustCompile(`^[a-z0-9_]{1,64}$`)
)

// Client talks to the Auth admin API. It is safe for concurrent use.
type Client struct {
	base    string  // e.g. https://abc.supabase.co/auth/v1
	key     *secret // pointer: value copies of Client print an address, not the key
	hc      *http.Client
	timeout time.Duration
}

// Option configures New.
type Option func(*Client)

// WithHTTPClient injects the HTTP client (for tests or custom transports).
// It is copied; its redirect policy is always overridden to not follow.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if hc != nil {
			cp := *hc
			c.hc = &cp
		}
	}
}

// WithTimeout sets the per-request timeout (default 5 seconds).
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		if d > 0 {
			c.timeout = d
		}
	}
}

// New builds a Client for the project URL (for example
// https://abc.supabase.co). The URL must be https unless the host is loopback,
// and must not carry credentials, query or fragment.
func New(baseURL, serviceRoleKey string, opts ...Option) (*Client, error) {
	if serviceRoleKey == "" || strings.ContainsAny(serviceRoleKey, " \r\n\t") {
		return nil, fmt.Errorf("%w: service role key", ErrInvalidArgument)
	}
	u, err := url.Parse(baseURL)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("%w: base URL", ErrInvalidArgument)
	}
	switch u.Scheme {
	case "https":
	case "http":
		if !isLoopback(u.Hostname()) {
			return nil, fmt.Errorf("%w: base URL must be https", ErrInvalidArgument)
		}
	default:
		return nil, fmt.Errorf("%w: base URL scheme", ErrInvalidArgument)
	}
	c := &Client{
		base:    strings.TrimRight(u.Scheme+"://"+u.Host+u.Path, "/") + "/auth/v1",
		key:     &secret{v: serviceRoleKey},
		timeout: DefaultTimeout,
	}
	for _, o := range opts {
		o(c)
	}
	if c.hc == nil {
		c.hc = &http.Client{}
	}
	c.hc.Timeout = c.timeout
	c.hc.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return c, nil
}

func isLoopback(h string) bool {
	if h == "localhost" {
		return true
	}
	ip := net.ParseIP(h)
	return ip != nil && ip.IsLoopback()
}

// String redacts the key.
func (c *Client) String() string { return "authadminx.Client{base: " + c.base + ", key: [REDACTED]}" }

// GoString redacts the key.
func (c *Client) GoString() string { return c.String() }

// Format redacts the key for every fmt verb, including %+v.
func (c *Client) Format(f fmt.State, _ rune) { _, _ = io.WriteString(f, c.String()) }

// Invite sends an invitation email, creating the user if needed.
func (c *Client) Invite(ctx context.Context, email string, opts InviteOptions) (*User, error) {
	if err := validEmail(email); err != nil {
		return nil, err
	}
	q := ""
	if opts.RedirectTo != "" {
		ru, err := url.Parse(opts.RedirectTo)
		if err != nil || (ru.Scheme != "https" && ru.Scheme != "http") || ru.Host == "" || ru.User != nil {
			return nil, fmt.Errorf("%w: redirect URL", ErrInvalidArgument)
		}
		q = "?redirect_to=" + url.QueryEscape(opts.RedirectTo)
	}
	body := map[string]any{"email": email}
	if opts.Data != nil {
		body["data"] = opts.Data
	}
	var u User
	if _, err := c.do(ctx, http.MethodPost, "/invite"+q, body, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

// ListUsers returns one page (page starts at 1, perPage 1..1000).
func (c *Client) ListUsers(ctx context.Context, page, perPage int) (*ListResult, error) {
	if page < 1 || perPage < 1 || perPage > maxPerPage {
		return nil, fmt.Errorf("%w: pagination", ErrInvalidArgument)
	}
	var out struct {
		Users []User `json:"users"`
	}
	h, err := c.do(ctx, http.MethodGet, "/admin/users?page="+strconv.Itoa(page)+"&per_page="+strconv.Itoa(perPage), nil, &out)
	if err != nil {
		return nil, err
	}
	total := -1
	if n, err := strconv.Atoi(h.Get("X-Total-Count")); err == nil && n >= 0 {
		total = n
	}
	return &ListResult{Users: out.Users, Total: total}, nil
}

// GetUser fetches a user by id.
func (c *Client) GetUser(ctx context.Context, id string) (*User, error) {
	if err := validID(id); err != nil {
		return nil, err
	}
	var u User
	if _, err := c.do(ctx, http.MethodGet, "/admin/users/"+id, nil, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

// UpdateUser changes the given fields of a user and returns the result.
func (c *Client) UpdateUser(ctx context.Context, id string, p UpdateUserParams) (*User, error) {
	if err := validID(id); err != nil {
		return nil, err
	}
	var u User
	if _, err := c.do(ctx, http.MethodPut, "/admin/users/"+id, p, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

// Ban blocks sign-in for the user (BanForever); existing access tokens stay
// valid until they expire.
func (c *Client) Ban(ctx context.Context, id string) (*User, error) {
	return c.UpdateUser(ctx, id, UpdateUserParams{BanDuration: BanForever})
}

// Unban lifts a ban.
func (c *Client) Unban(ctx context.Context, id string) (*User, error) {
	return c.UpdateUser(ctx, id, UpdateUserParams{BanDuration: BanNone})
}

// DeleteUser removes a user; soft keeps the row but anonymises it.
func (c *Client) DeleteUser(ctx context.Context, id string, soft bool) error {
	if err := validID(id); err != nil {
		return err
	}
	_, err := c.do(ctx, http.MethodDelete, "/admin/users/"+id, map[string]bool{"should_soft_delete": soft}, nil)
	return err
}

func validID(id string) error {
	if !uuidRe.MatchString(id) {
		return fmt.Errorf("%w: user id", ErrInvalidArgument)
	}
	return nil
}

func validEmail(e string) error {
	if len(e) > 254 || strings.ContainsAny(e, "\r\n<>") {
		return fmt.Errorf("%w: email", ErrInvalidArgument)
	}
	a, err := mail.ParseAddress(e)
	if err != nil || a.Address != e || a.Name != "" {
		return fmt.Errorf("%w: email", ErrInvalidArgument)
	}
	return nil
}

func (c *Client) do(ctx context.Context, method, path string, in, out any) (http.Header, error) {
	var rdr io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return nil, fmt.Errorf("%w: body", ErrInvalidArgument)
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, rdr)
	if err != nil {
		return nil, fmt.Errorf("%w: request", ErrInvalidArgument)
	}
	req.Header.Set("Authorization", "Bearer "+c.key.v)
	req.Header.Set("apikey", c.key.v)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		// Drop the *url.Error wrapper: it embeds the URL. Keep ctx causes matchable.
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err
		}
		return nil, fmt.Errorf("authadminx: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("authadminx: read response: %w", err)
	}
	if len(data) > MaxResponseBytes {
		return nil, ErrResponseTooLarge
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		var e struct {
			ErrorCode string `json:"error_code"`
		}
		_ = json.Unmarshal(data, &e)
		if !codeRe.MatchString(e.ErrorCode) {
			e.ErrorCode = ""
		}
		return nil, newAPIError(resp.StatusCode, e.ErrorCode)
	}
	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			return nil, ErrInvalidResponse
		}
	}
	return resp.Header, nil
}

// secret holds the service role key; it redacts itself under every fmt verb.
type secret struct{ v string }

func (*secret) String() string   { return "[REDACTED]" }
func (*secret) GoString() string { return "[REDACTED]" }
