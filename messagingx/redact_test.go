package messagingx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"syscall"
	"testing"
	"time"
)

// secretToken looks like a real bot token; it must never appear in an error.
const secretToken = "123456789:AAF-secret_Token-xyz"

// roundTripFunc is a fake transport.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// echoTimeoutError is a net.Error timeout whose text echoes the request URL.
type echoTimeoutError struct{ msg string }

func (e *echoTimeoutError) Error() string   { return e.msg }
func (e *echoTimeoutError) Timeout() bool   { return true }
func (e *echoTimeoutError) Temporary() bool { return true }

// dialClient returns a client whose dialer fails with err, so the real
// net/http stack wraps it in *url.Error carrying the token-bearing URL.
func dialClient(err error) *http.Client {
	return &http.Client{Transport: &http.Transport{
		DialContext: func(context.Context, string, string) (net.Conn, error) { return nil, err },
	}}
}

// assertNoLeak checks every rendering of err, and of every error in its chain.
func assertNoLeak(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error")
	}
	var chain []error
	for e := err; e != nil; e = errors.Unwrap(e) {
		chain = append(chain, e)
	}
	for i, e := range chain {
		for _, verb := range []string{"%v", "%+v", "%#v", "%s", "%q"} {
			out := fmt.Sprintf(verb, e)
			for _, bad := range []string{secretToken, url.PathEscape(secretToken), "secret_Token"} {
				if strings.Contains(out, bad) {
					t.Errorf("chain[%d] %s = %q leaks %q", i, verb, out, bad)
				}
			}
			if strings.Count(out, "/bot") != strings.Count(out, "/bot<redacted>") {
				t.Errorf("chain[%d] %s = %q shows an unredacted /bot path", i, verb, out)
			}
		}
	}
	var ue *url.Error
	if errors.As(err, &ue) {
		t.Errorf("errors.As(*url.Error) matched %q; the URL holds the token", ue.URL)
	}
}

// assertClassified checks that redaction kept the error usable.
func assertClassified(t *testing.T, err, wantIs error, timeout bool, wantText string) {
	t.Helper()
	if wantIs != nil && !errors.Is(err, wantIs) {
		t.Errorf("errors.Is(%v, %v) = false", err, wantIs)
	}
	if timeout {
		var ne net.Error
		if !errors.As(err, &ne) || !ne.Timeout() {
			t.Errorf("errors.As(net.Error).Timeout() = false for %v", err)
		}
	}
	if !strings.HasPrefix(err.Error(), "messagingx: telegram ") {
		t.Errorf("error %q lacks the package prefix", err)
	}
	if wantText != "" && !strings.Contains(err.Error(), wantText) {
		t.Errorf("error %q does not contain %q", err, wantText)
	}
}

func TestTelegram_TransportErrorsRedactToken(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	}))
	defer slow.Close()
	stall := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"ok":`)
		w.(http.Flusher).Flush()
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	}))
	defer stall.Close()

	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name     string
		baseURL  string
		client   *http.Client
		ctx      func() (context.Context, context.CancelFunc)
		wantIs   error // nil: none required
		timeout  bool  // net.Error Timeout() must be true
		wantText string
	}{
		{
			name:     "dial refused",
			client:   dialClient(&net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}),
			wantIs:   syscall.ECONNREFUSED,
			wantText: "send request",
		},
		{
			name:     "dns failure",
			client:   dialClient(&net.DNSError{Err: "no such host", Name: "api.telegram.invalid", IsNotFound: true}),
			wantText: "no such host",
		},
		{
			name:     "client timeout awaiting headers",
			baseURL:  slow.URL,
			client:   &http.Client{Timeout: 50 * time.Millisecond},
			wantIs:   context.DeadlineExceeded,
			timeout:  true,
			wantText: "send request",
		},
		{
			name:     "client timeout reading body",
			baseURL:  stall.URL,
			client:   &http.Client{Timeout: 100 * time.Millisecond},
			wantIs:   context.DeadlineExceeded,
			timeout:  true,
			wantText: "decode response",
		},
		{
			name:    "context deadline",
			baseURL: slow.URL,
			client:  &http.Client{},
			ctx: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 50*time.Millisecond)
			},
			wantIs:  context.DeadlineExceeded,
			timeout: true,
		},
		{
			name:    "context canceled",
			baseURL: slow.URL,
			client:  &http.Client{},
			ctx:     func() (context.Context, context.CancelFunc) { return canceled, func() {} },
			wantIs:  context.Canceled,
		},
		{
			name: "transport echoes the URL",
			client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return nil, fmt.Errorf("proxy refused %s (path %s): %w", r.URL, r.URL.EscapedPath(), context.DeadlineExceeded)
			})},
			wantIs:   context.DeadlineExceeded,
			timeout:  true,
			wantText: "<redacted>",
		},
		{
			name: "transport echoes the URL in a net.Error timeout",
			client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return nil, &echoTimeoutError{msg: "i/o timeout on " + r.URL.String()}
			})},
			timeout:  true,
			wantText: "i/o timeout on",
		},
		{
			name: "transport echoes the URL without a sentinel",
			client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return nil, errors.New("upstream rejected " + r.URL.String())
			})},
			wantText: "upstream rejected",
		},
		{
			name:     "invalid base URL",
			baseURL:  "http://bad host",
			client:   &http.Client{},
			wantText: "build request",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := tt.baseURL
			if base == "" {
				base = "http://api.telegram.test"
			}
			tg := NewTelegram(secretToken, WithBaseURL(base), WithHTTPClient(tt.client))
			ctx, cancel := context.Background(), context.CancelFunc(func() {})
			if tt.ctx != nil {
				ctx, cancel = tt.ctx()
			}
			defer cancel()

			errs := map[string]error{"Send": tg.Send(ctx, "-100", "halo")}
			_, next, pollErr := tg.Poll(ctx, 7)
			errs["Poll"] = pollErr
			if next != 7 {
				t.Errorf("Poll next offset = %d on error, want 7", next)
			}
			for call, err := range errs {
				t.Run(call, func(t *testing.T) {
					assertNoLeak(t, err)
					assertClassified(t, err, tt.wantIs, tt.timeout, tt.wantText)
				})
			}
		})
	}
}

func TestTelegram_APIErrorsRedactDescription(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		want    string
		notWant []string
	}{
		{
			name:   "known description kept",
			status: http.StatusBadRequest,
			body:   `{"ok":false,"error_code":400,"description":"Bad Request: chat not found"}`,
			want:   "messagingx: telegram sendMessage: api error 400 Bad Request: chat not found",
		},
		{
			name:    "unknown description replaced by status text",
			status:  http.StatusBadRequest,
			body:    `{"ok":false,"error_code":400,"description":"Bad Request: can't parse entities in 'nomor rekening 1234567890'"}`,
			want:    "messagingx: telegram sendMessage: api error 400 Bad Request",
			notWant: []string{"1234567890", "parse entities"},
		},
		{
			name:    "description echoing the token",
			status:  http.StatusUnauthorized,
			body:    `{"ok":false,"error_code":401,"description":"Unauthorized: invalid token ` + secretToken + `"}`,
			want:    "messagingx: telegram sendMessage: api error 401 Unauthorized",
			notWant: []string{"invalid token"},
		},
		{
			name:   "retry after",
			status: http.StatusTooManyRequests,
			body:   `{"ok":false,"error_code":429,"description":"Too Many Requests: retry after 5","parameters":{"retry_after":5}}`,
			want:   "messagingx: telegram sendMessage: api error 429 Too Many Requests (retry after 5s)",
		},
		{
			name:   "no error_code uses HTTP status",
			status: http.StatusOK,
			body:   `{"ok":false}`,
			want:   "messagingx: telegram sendMessage: api error 200 OK",
		},
		{
			name:    "non-JSON error page",
			status:  http.StatusBadGateway,
			body:    `<html>bad gateway for /bot` + secretToken + `/sendMessage</html>`,
			want:    "messagingx: telegram sendMessage: api error 502 Bad Gateway",
			notWant: []string{"html"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = io.WriteString(w, tt.body)
			}))
			defer srv.Close()
			tg := NewTelegram(secretToken, WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
			err := tg.Send(context.Background(), "-100", "halo")
			assertNoLeak(t, err)
			if err.Error() != tt.want {
				t.Errorf("error = %q, want %q", err, tt.want)
			}
			for _, verb := range []string{"%v", "%+v", "%#v"} {
				out := fmt.Sprintf(verb, err)
				for _, bad := range tt.notWant {
					if strings.Contains(out, bad) {
						t.Errorf("%s = %q contains %q", verb, out, bad)
					}
				}
			}
		})
	}
}

func TestTelegram_DecodeErrorsRedacted(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "malformed response", body: `{"ok":tru`, want: "messagingx: telegram getUpdates: decode response: "},
		{name: "malformed result", body: `{"ok":true,"result":{"not":"a list"}}`, want: "messagingx: telegram getUpdates: decode updates: "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, tt.body)
			}))
			defer srv.Close()
			tg := NewTelegram(secretToken, WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
			_, _, err := tg.Poll(context.Background(), 0)
			assertNoLeak(t, err)
			if !strings.HasPrefix(err.Error(), tt.want) {
				t.Errorf("error = %q, want prefix %q", err, tt.want)
			}
		})
	}
}

func TestNewTelegram_DefaultClient(t *testing.T) {
	tests := []struct {
		name string
		opts []Option
		want time.Duration
	}{
		{name: "default timeout", want: defaultTimeout},
		{name: "nil client keeps default", opts: []Option{WithHTTPClient(nil)}, want: defaultTimeout},
		{name: "custom client", opts: []Option{WithHTTPClient(&http.Client{Timeout: 3 * time.Second})}, want: 3 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tg := NewTelegram(secretToken, tt.opts...)
			if tg.hc == nil || tg.hc.Timeout != tt.want {
				t.Errorf("client timeout = %v, want %v", tg.hc.Timeout, tt.want)
			}
		})
	}
	if defaultTimeout <= 0 || defaultTimeout > 30*time.Second {
		t.Errorf("defaultTimeout = %v, want a bound in (0, 30s]", defaultTimeout)
	}
}

func TestRedact(t *testing.T) {
	plain := errors.New("plain")
	tests := []struct {
		name    string
		err     error
		token   string
		wantMsg string
		same    bool // the cause is returned unchanged
	}{
		{name: "no token in message", err: plain, token: secretToken, wantMsg: "plain", same: true},
		{name: "empty token never replaces", err: errors.New("x"), token: "", wantMsg: "x"},
		{name: "path-escaped token", err: errors.New("GET /bot" + url.PathEscape("a b") + "/x"), token: "a b", wantMsg: "GET /bot<redacted>/x"},
		{name: "query-escaped token", err: errors.New("q=" + url.QueryEscape("a b")), token: "a b", wantMsg: "q=<redacted>"},
		{name: "url.Error unwrapped", err: &url.Error{Op: "Post", URL: "https://x/bot" + secretToken, Err: plain}, token: secretToken, wantMsg: "plain"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := redact(tt.err, tt.token)
			if got.Error() != tt.wantMsg {
				t.Errorf("redact = %q, want %q", got, tt.wantMsg)
			}
			if tt.same && !errors.Is(got, tt.err) {
				t.Errorf("redact replaced an error without the token")
			}
		})
	}
}
