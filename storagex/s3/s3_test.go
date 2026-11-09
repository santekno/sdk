package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/santekno/sdk/storagex"
)

// onlyReader hides Seek so Put has to spool the body.
type onlyReader struct{ r io.Reader }

func (o onlyReader) Read(p []byte) (int, error) { return o.r.Read(p) }

func TestNewValidation(t *testing.T) {
	base := Config{Endpoint: "https://proj.supabase.co/storage/v1/s3", Region: "ap-southeast-1", Bucket: "media", AccessKeyID: testAKID, SecretAccessKey: testSecret}
	bad := map[string]func(c *Config){
		"empty endpoint":     func(c *Config) { c.Endpoint = "" },
		"relative endpoint":  func(c *Config) { c.Endpoint = "/storage/v1/s3" },
		"ftp endpoint":       func(c *Config) { c.Endpoint = "ftp://proj.supabase.co" },
		"plain http remote":  func(c *Config) { c.Endpoint = "http://proj.supabase.co/storage/v1/s3" },
		"user info":          func(c *Config) { c.Endpoint = "https://" + testAKID + ":" + testSecret + "@proj.supabase.co/s3" },
		"query":              func(c *Config) { c.Endpoint = "https://proj.supabase.co/s3?token=" + testSecret },
		"fragment":           func(c *Config) { c.Endpoint = "https://proj.supabase.co/s3#x" },
		"unparsable":         func(c *Config) { c.Endpoint = "https://proj supabase.co:bad" },
		"empty region":       func(c *Config) { c.Region = "" },
		"region slash":       func(c *Config) { c.Region = "us/east" },
		"empty bucket":       func(c *Config) { c.Bucket = "" },
		"bucket slash":       func(c *Config) { c.Bucket = "a/b" },
		"bucket dotdot":      func(c *Config) { c.Bucket = ".." },
		"empty access key":   func(c *Config) { c.AccessKeyID = "" },
		"access key slash":   func(c *Config) { c.AccessKeyID = "AKIA/" + testSecret },
		"access key space":   func(c *Config) { c.AccessKeyID = "AKIA " },
		"empty secret":       func(c *Config) { c.SecretAccessKey = "" },
		"overlong region":    func(c *Config) { c.Region = strings.Repeat("r", 300) },
		"overlong bucket":    func(c *Config) { c.Bucket = strings.Repeat("b", 256) },
		"non-ascii bucket":   func(c *Config) { c.Bucket = "médïa" },
		"loopback lookalike": func(c *Config) { c.Endpoint = "http://localhost.evil.test/s3" },
	}
	for name, mutate := range bad {
		t.Run(name, func(t *testing.T) {
			c := base
			mutate(&c)
			_, err := New(c)
			if !errors.Is(err, ErrInvalidConfig) {
				t.Fatalf("err = %v, want ErrInvalidConfig", err)
			}
			for _, secret := range []string{testSecret, testAKID, "proj.supabase.co", "evil"} {
				if strings.Contains(err.Error(), secret) {
					t.Errorf("error leaks %q: %v", secret, err)
				}
			}
		})
	}
	good := map[string]string{
		"https://proj.supabase.co/storage/v1/s3":  "https://proj.supabase.co/storage/v1/s3/media/k",
		"https://proj.supabase.co:443/s3/":        "https://proj.supabase.co/s3/media/k",
		"https://minio.test:9000":                 "https://minio.test:9000/media/k",
		"http://127.0.0.1:54321/storage/v1/s3":    "http://127.0.0.1:54321/storage/v1/s3/media/k",
		"http://localhost:54321/storage/v1/s3":    "http://localhost:54321/storage/v1/s3/media/k",
		"http://[::1]:9000":                       "http://[::1]:9000/media/k",
		"http://127.0.0.1:80/s3":                  "http://127.0.0.1/s3/media/k",
		"HTTPS://Proj.Supabase.CO/storage/v1/s3/": "https://proj.supabase.co/storage/v1/s3/media/k",
	}
	for endpoint, want := range good {
		c := base
		c.Endpoint = endpoint
		cl, err := New(c)
		if err != nil {
			t.Errorf("New(%q): %v", endpoint, err)
			continue
		}
		if got := cl.objectURL("k"); got != want {
			t.Errorf("objectURL for %q = %q, want %q", endpoint, got, want)
		}
	}
}

const specialKey = "u/a b+c$&=,;:@!*'()~ü.jpg"

var roundTripCases = []struct {
	key  string
	body func() io.Reader
	want string
}{
	{"v/0123456789abcdef-480.webp", func() io.Reader { return bytes.NewReader([]byte("webp-bytes")) }, "webp-bytes"},
	{specialKey, func() io.Reader { return strings.NewReader("special") }, "special"},
	{"backups/2026-11-09/db.dump.age", func() io.Reader { return onlyReader{strings.NewReader("spooled body")} }, "spooled body"},
	{"empty/object", func() io.Reader { return strings.NewReader("") }, ""},
	{"empty/unknown-length", func() io.Reader { return onlyReader{strings.NewReader("")} }, ""},
	{"nil/reader", func() io.Reader { return nil }, ""},
	{"v/" + strings.Repeat("ab/", 50) + "x", func() io.Reader { return strings.NewReader("deep") }, "deep"},
}

func TestRoundTrip(t *testing.T) {
	_, srv := newFake(t)
	c := newTestClient(t, srv)
	for _, tc := range roundTripCases {
		t.Run(tc.key, func(t *testing.T) {
			checkRoundTrip(t, c, srv.URL, tc.key, tc.body(), tc.want)
		})
	}
	// Temporary spool files are removed.
	if entries, _ := os.ReadDir(c.tempDir); len(entries) != 0 {
		t.Errorf("temp files left: %v", entries)
	}
}

func checkRoundTrip(t *testing.T, c *Client, base, key string, body io.Reader, want string) {
	t.Helper()
	ctx := context.Background()
	u, err := c.Put(ctx, key, "image/webp", body)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if wantURL := base + testBase + "/" + testBucket + "/" + uriEncode(key, true); u != wantURL {
		t.Errorf("Put URL = %q, want %q", u, wantURL)
	}
	rc, err := c.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()
	if string(got) != want {
		t.Errorf("Get = %q, want %q", got, want)
	}
	info, err := c.Stat(ctx, key)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	wantInfo := storagex.ObjectInfo{Key: key, Size: int64(len(want)), ContentType: "image/webp", ETag: info.ETag, LastModified: testTime}
	if info != wantInfo || len(info.ETag) != 32 || strings.Contains(info.ETag, `"`) {
		t.Errorf("Stat = %+v", info)
	}
	if err := c.Delete(ctx, key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := c.Stat(ctx, key); !errors.Is(err, storagex.ErrNotFound) {
		t.Errorf("Stat after Delete: %v", err)
	}
}

func TestWireFormat(t *testing.T) {
	f, srv := newFake(t)
	c := newTestClient(t, srv)
	if _, err := c.Put(context.Background(), specialKey, "image/jpeg", strings.NewReader("x")); err != nil {
		t.Fatal(err)
	}
	// The wire path is the single-encoded SigV4 canonical path.
	if want := "PUT " + testBase + "/" + testBucket + "/u/a%20b%2Bc%24%26%3D%2C%3B%3A%40%21%2A%27%28%29~%C3%BC.jpg"; f.requests[0] != want {
		t.Errorf("request line = %q, want %q", f.requests[0], want)
	}
	h := f.headers[0]
	if h.Get("Accept-Encoding") != "identity" {
		t.Errorf("Accept-Encoding = %q, want identity", h.Get("Accept-Encoding"))
	}
	// Server calls sign the body hash; UNSIGNED-PAYLOAD is only for presigning.
	if sum := h.Get("X-Amz-Content-Sha256"); sum != sha256Hex([]byte("x")) {
		t.Errorf("X-Amz-Content-Sha256 = %q, want the body hash", sum)
	}
}

func TestNotFound(t *testing.T) {
	_, srv := newFake(t)
	c := newTestClient(t, srv)
	ctx := context.Background()
	_, err := c.Get(ctx, specialKey)
	var se *Error
	if !errors.Is(err, storagex.ErrNotFound) || !errors.As(err, &se) || se.StatusCode != 404 || se.Code != "NoSuchKey" || se.Op != "get" {
		t.Errorf("Get missing: %v (%+v)", err, se)
	}
	if want := "storagex/s3: get: object not found (status 404 NoSuchKey)"; err.Error() != want {
		t.Errorf("error = %q, want %q", err, want)
	}
	if _, err := c.Stat(ctx, "missing/key"); !errors.Is(err, storagex.ErrNotFound) {
		t.Errorf("Stat missing: %v", err)
	}
	if err := c.Delete(ctx, "missing/key"); err != nil {
		t.Errorf("Delete missing: %v", err)
	}
}

func TestPutSeekableFromCurrentOffset(t *testing.T) {
	_, srv := newFake(t)
	c := newTestClient(t, srv)
	r := bytes.NewReader([]byte("skipBODY"))
	_, _ = r.Read(make([]byte, 4))
	if _, err := c.Put(context.Background(), "k", "text/plain", r); err != nil {
		t.Fatal(err)
	}
	rc, err := c.Get(context.Background(), "k")
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()
	if b, _ := io.ReadAll(rc); string(b) != "BODY" {
		t.Errorf("body = %q, want BODY", b)
	}
}

func TestPutDefaultContentTypeAndInvalidInput(t *testing.T) {
	f, srv := newFake(t)
	c := newTestClient(t, srv)
	ctx := context.Background()
	if _, err := c.Put(ctx, "k", "", strings.NewReader("x")); err != nil {
		t.Fatal(err)
	}
	if ct := f.objects["k"].contentType; ct != "application/octet-stream" {
		t.Errorf("content type = %q", ct)
	}
	if _, err := c.Put(ctx, "k", "text/plain\r\nX-Evil: 1", strings.NewReader("x")); !errors.Is(err, ErrInvalidRequest) {
		t.Errorf("header injection: err = %v", err)
	}
	for _, key := range []string{"", "../x", "/abs", "a//b", `a\b`} {
		if _, err := c.Put(ctx, key, "", strings.NewReader("x")); !errors.Is(err, storagex.ErrInvalidKey) {
			t.Errorf("Put(%q): %v", key, err)
		}
		if _, err := c.Get(ctx, key); !errors.Is(err, storagex.ErrInvalidKey) {
			t.Errorf("Get(%q): %v", key, err)
		}
		if _, err := c.Stat(ctx, key); !errors.Is(err, storagex.ErrInvalidKey) {
			t.Errorf("Stat(%q): %v", key, err)
		}
		if err := c.Delete(ctx, key); !errors.Is(err, storagex.ErrInvalidKey) {
			t.Errorf("Delete(%q): %v", key, err)
		}
		if _, err := c.PresignPut(ctx, key, "image/png", time.Minute); !errors.Is(err, storagex.ErrInvalidKey) {
			t.Errorf("PresignPut(%q): %v", key, err)
		}
	}
	if _, err := c.List(ctx, storagex.ListOptions{Prefix: "../"}); !errors.Is(err, storagex.ErrInvalidKey) {
		t.Errorf("List bad prefix: %v", err)
	}
	if n := len(f.requests); n != 1 {
		t.Errorf("invalid input reached the server: %d requests", n)
	}
}

func TestPutReaderErrors(t *testing.T) {
	_, srv := newFake(t)
	c := newTestClient(t, srv)
	boom := errors.New("reader broke")
	if _, err := c.Put(context.Background(), "k", "", onlyReader{iotest.ErrReader(boom)}); !errors.Is(err, boom) {
		t.Errorf("spool error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.Put(ctx, "k", "", onlyReader{strings.NewReader("x")}); !errors.Is(err, context.Canceled) {
		t.Errorf("canceled spool = %v", err)
	}
	if _, err := c.Put(ctx, "k", "", strings.NewReader("x")); !errors.Is(err, context.Canceled) {
		t.Errorf("canceled seekable = %v", err)
	}
	if entries, _ := os.ReadDir(c.tempDir); len(entries) != 0 {
		t.Errorf("temp files left: %v", entries)
	}
	bad := Config{Endpoint: srv.URL, Region: testRegion, Bucket: testBucket, AccessKeyID: testAKID, SecretAccessKey: testSecret, TempDir: "/nonexistent/dir/for/storagex"}
	c2, err := New(bad)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c2.Put(context.Background(), "k", "", onlyReader{strings.NewReader("x")}); err == nil {
		t.Error("Put with unusable TempDir: want error")
	}
}

// seekFailer reports a seekable type but fails the seek back to the start.
type seekFailer struct {
	*strings.Reader
	calls int
}

func (s *seekFailer) Seek(off int64, whence int) (int64, error) {
	s.calls++
	if s.calls >= 3 {
		return 0, errors.New("seek failed")
	}
	return s.Reader.Seek(off, whence)
}

func TestPutSeekFailure(t *testing.T) {
	_, srv := newFake(t)
	c := newTestClient(t, srv)
	if _, err := c.Put(context.Background(), "k", "", &seekFailer{Reader: strings.NewReader("x")}); err == nil {
		t.Error("want error when seeking back fails")
	}
}

func TestNoOverwrite(t *testing.T) {
	f, srv := newFake(t)
	ctx := context.Background()
	plain := newTestClient(t, srv)
	guarded := newTestClient(t, srv, storagex.WithNoOverwrite())

	if _, err := guarded.Put(ctx, "v/h-480.webp", "image/webp", strings.NewReader("one")); err != nil {
		t.Fatal(err)
	}
	if got := f.headers[len(f.headers)-1].Get("If-None-Match"); got != "*" {
		t.Errorf("If-None-Match = %q, want *", got)
	}
	_, err := guarded.Put(ctx, "v/h-480.webp", "image/webp", strings.NewReader("two"))
	var se *Error
	if !errors.Is(err, storagex.ErrAlreadyExists) || !errors.As(err, &se) || se.StatusCode != 412 {
		t.Fatalf("second Put: %v", err)
	}
	if want := "storagex/s3: put: object already exists (status 412 PreconditionFailed)"; err.Error() != want {
		t.Errorf("error = %q, want %q", err, want)
	}
	if string(f.objects["v/h-480.webp"].body) != "one" {
		t.Error("object overwritten")
	}
	if _, err := plain.Put(ctx, "v/h-480.webp", "image/webp", strings.NewReader("three")); err != nil {
		t.Fatal(err)
	}
	if got := f.headers[len(f.headers)-1].Get("If-None-Match"); got != "" {
		t.Errorf("plain Put sent If-None-Match %q", got)
	}
	if string(f.objects["v/h-480.webp"].body) != "three" {
		t.Error("plain Put did not overwrite")
	}
}

func seedList(t *testing.T) (*fakeS3, *Client) {
	t.Helper()
	f, srv := newFake(t)
	c := newTestClient(t, srv)
	for _, k := range []string{"u/1", "u/2", "u/3", "v/1", "u0"} {
		if _, err := c.Put(context.Background(), k, "", strings.NewReader(k)); err != nil {
			t.Fatal(err)
		}
	}
	return f, c
}

func listKeys(t *testing.T, c *Client, opts storagex.ListOptions) (string, string) {
	t.Helper()
	page, err := c.List(context.Background(), opts)
	if err != nil {
		t.Fatalf("List(%+v): %v", opts, err)
	}
	return fmt.Sprint(keys(page)), page.NextToken
}

func TestListPages(t *testing.T) {
	_, c := seedList(t)
	got, token := listKeys(t, c, storagex.ListOptions{Prefix: "u/", Limit: 2})
	if got != "[u/1 u/2]" || token == "" {
		t.Fatalf("page 1 = %v token %q", got, token)
	}
	got, token = listKeys(t, c, storagex.ListOptions{Prefix: "u/", Limit: 2, Token: token})
	if got != "[u/3]" || token != "" {
		t.Errorf("page 2 = %v token %q", got, token)
	}
	page, err := c.List(context.Background(), storagex.ListOptions{Prefix: "u/1"})
	if err != nil || len(page.Objects) != 1 {
		t.Fatalf("List = %+v, %v", page, err)
	}
	want := storagex.ObjectInfo{Key: "u/1", Size: 3, ETag: "etag-u/1", LastModified: testTime}
	if page.Objects[0] != want {
		t.Errorf("object = %+v, want %+v", page.Objects[0], want)
	}
}

func TestListDefaultsAndShortPages(t *testing.T) {
	f, c := seedList(t)
	if got, _ := listKeys(t, c, storagex.ListOptions{}); got != "[u/1 u/2 u/3 u0 v/1]" {
		t.Errorf("List all = %v", got)
	}
	last := f.requests[len(f.requests)-1]
	if !strings.Contains(last, "max-keys=1000") || strings.Contains(last, "prefix=") {
		t.Errorf("default list request = %s", last)
	}
	// The server may return fewer keys than asked.
	f.listPageCap = 1
	if got, token := listKeys(t, c, storagex.ListOptions{Prefix: "u"}); got != "[u/1]" || token == "" {
		t.Errorf("capped page = %v %q", got, token)
	}
}
func keys(p storagex.ListPage) []string {
	out := make([]string, len(p.Objects))
	for i, o := range p.Objects {
		out[i] = o.Key
	}
	return out
}

func TestListMalformedResponses(t *testing.T) {
	f, srv := newFake(t)
	c := newTestClient(t, srv)
	cases := map[string]string{
		"truncated without token": `<ListBucketResult><IsTruncated>true</IsTruncated></ListBucketResult>`,
		"same token again":        `<ListBucketResult><IsTruncated>true</IsTruncated><NextContinuationToken>tok</NextContinuationToken></ListBucketResult>`,
		"not xml":                 `{"objects": []}`,
		"too large":               "<ListBucketResult>" + strings.Repeat("<x/>", maxListBody/4) + "</ListBucketResult>",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			f.override = func(w http.ResponseWriter, _ *http.Request) bool {
				_, _ = io.WriteString(w, body)
				return true
			}
			_, err := c.List(context.Background(), storagex.ListOptions{Token: "tok"})
			var se *Error
			if !errors.As(err, &se) || se.StatusCode != 200 || se.Op != "list" {
				t.Errorf("err = %v", err)
			}
			if name == "too large" && (err == nil || !strings.HasSuffix(err.Error(), "response too large")) {
				t.Errorf("too large: err = %v", err)
			}
		})
	}
	f.override = func(w http.ResponseWriter, _ *http.Request) bool {
		s3Error(w, 403, "AccessDenied")
		return true
	}
	if _, err := c.List(context.Background(), storagex.ListOptions{}); err == nil || err.Error() != "storagex/s3: list: status 403 AccessDenied" {
		t.Errorf("list 403 = %v", err)
	}
}

func TestStatusErrorsAreRedacted(t *testing.T) {
	f, srv := newFake(t)
	c := newTestClient(t, srv)
	ctx := context.Background()
	const key = "u/secret-object-name.jpg"
	responses := map[string]func(w http.ResponseWriter, r *http.Request){
		"403 xml": func(w http.ResponseWriter, _ *http.Request) { s3Error(w, 403, "SignatureDoesNotMatch") },
		"500 html": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
			fmt.Fprintf(w, "<html>%s %s</html>", r.URL, r.Header.Get("Authorization"))
		},
		"bad code": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(400)
			fmt.Fprintf(w, "<Error><Code>%s</Code></Error>", r.URL)
		},
		"redirect": func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "https://attacker.test/collect?"+r.URL.RawQuery, http.StatusTemporaryRedirect)
		},
	}
	for name, respond := range responses {
		t.Run(name, func(t *testing.T) {
			f.override = func(w http.ResponseWriter, r *http.Request) bool { respond(w, r); return true }
			f.requests = nil
			ops := map[string]func() error{
				"put":    func() error { _, err := c.Put(ctx, key, "image/jpeg", strings.NewReader("x")); return err },
				"get":    func() error { _, err := c.Get(ctx, key); return err },
				"stat":   func() error { _, err := c.Stat(ctx, key); return err },
				"delete": func() error { return c.Delete(ctx, key) },
				"list":   func() error { _, err := c.List(ctx, storagex.ListOptions{Prefix: "u/secret"}); return err },
			}
			for op, call := range ops {
				err := call()
				var se *Error
				if !errors.As(err, &se) || se.StatusCode == 0 || se.Op != op {
					t.Errorf("%s: err = %v", op, err)
					continue
				}
				if errors.Is(err, storagex.ErrNotFound) || errors.Is(err, ErrTransport) {
					t.Errorf("%s: wrong sentinel: %v", op, err)
				}
				assertRedacted(t, err, "secret-object-name", srv.URL, "127.0.0.1", testAKID, testSecret, "Signature=", "attacker")
				if name == "bad code" && se.Code != "" {
					t.Errorf("unsafe code kept: %q", se.Code)
				}
			}
			if len(f.requests) != len(ops) {
				t.Errorf("redirect followed: %d requests for %d calls", len(f.requests), len(ops))
			}
		})
	}
}

// echoTransport fails every request with an error that embeds everything
// sensitive about it, like a misbehaving RoundTripper or proxy.
type echoTransport struct {
	leak    string
	timeout bool
}

type echoErr struct {
	msg     string
	timeout bool
}

func (e echoErr) Error() string   { return e.msg }
func (e echoErr) Timeout() bool   { return e.timeout }
func (e echoErr) Temporary() bool { return false }

func (t echoTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return nil, echoErr{msg: fmt.Sprintf("dial %s failed: auth=%q leak=%s", r.URL, r.Header.Get("Authorization"), t.leak), timeout: t.timeout}
}

func TestTransportErrorsAreRedacted(t *testing.T) {
	_, srv := newFake(t)
	presigner := newTestClient(t, srv)
	presigned, err := presigner.PresignPut(context.Background(), "u/upload-target.jpg", "image/jpeg", 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	sig := presigned.URL[strings.Index(presigned.URL, "X-Amz-Signature=")+len("X-Amz-Signature="):]
	secrets := []string{"secret-object-name", "upload-target", presigned.URL, sig, "X-Amz-", "Signature=", testAKID, testSecret, "proj.example"}

	for _, timeout := range []bool{false, true} {
		cfg := Config{
			Endpoint: "https://proj.example/storage/v1/s3", Region: testRegion, Bucket: testBucket,
			AccessKeyID: testAKID, SecretAccessKey: testSecret,
			HTTPClient: &http.Client{Transport: echoTransport{leak: presigned.URL, timeout: timeout}},
		}
		c, err := New(cfg)
		if err != nil {
			t.Fatal(err)
		}
		ops := map[string]func() error{
			"put": func() error {
				_, err := c.Put(context.Background(), "u/secret-object-name", "", strings.NewReader("x"))
				return err
			},
			"get":    func() error { _, err := c.Get(context.Background(), "u/secret-object-name"); return err },
			"stat":   func() error { _, err := c.Stat(context.Background(), "u/secret-object-name"); return err },
			"delete": func() error { return c.Delete(context.Background(), "u/secret-object-name") },
			"list": func() error {
				_, err := c.List(context.Background(), storagex.ListOptions{Prefix: "u/secret-object-name"})
				return err
			},
		}
		for op, call := range ops {
			err := call()
			var se *Error
			if !errors.As(err, &se) || se.StatusCode != 0 || !errors.Is(err, ErrTransport) || se.Timeout() != timeout {
				t.Errorf("%s (timeout=%v): err = %#v", op, timeout, err)
			}
			if errors.As(err, new(*url.Error)) {
				t.Errorf("%s: *url.Error kept in chain", op)
			}
			assertRedacted(t, err, secrets...)
		}
	}
}

func TestTransportContextErrors(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-block }))
	defer srv.Close()
	defer close(block)
	c := newTestClient(t, srv)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := c.Get(ctx, "u/secret-object-name")
	var se *Error
	if !errors.Is(err, context.DeadlineExceeded) || !errors.Is(err, ErrTransport) || !errors.As(err, &se) || !se.Timeout() {
		t.Errorf("deadline: %v", err)
	}
	if err.Error() != "storagex/s3: get: transport error: deadline exceeded" {
		t.Errorf("deadline message = %q", err)
	}

	ctx, cancel2 := context.WithCancel(context.Background())
	go func() { time.Sleep(20 * time.Millisecond); cancel2() }()
	err = c.Delete(ctx, "u/secret-object-name")
	if !errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("cancel: %v", err)
	}
	assertRedacted(t, err, "secret-object-name", srv.URL)
}

// assertRedacted checks every fmt rendering of err and of each error in its
// Unwrap chain.
func assertRedacted(t *testing.T, err error, secrets ...string) {
	t.Helper()
	for e := err; e != nil; e = errors.Unwrap(e) {
		for _, verb := range []string{"%v", "%+v", "%#v", "%s", "%q"} {
			out := fmt.Sprintf(verb, e)
			for _, s := range secrets {
				if s != "" && strings.Contains(out, s) {
					t.Errorf("%s of %T leaks %q: %s", verb, e, s, out)
				}
			}
		}
	}
}

func TestPresignPutURL(t *testing.T) {
	_, srv := newFake(t)
	c := newTestClient(t, srv)
	p, err := c.PresignPut(context.Background(), "u/0b5c-uuid", "image/jpeg", 10*time.Minute+500*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if p.Method != http.MethodPut || !p.ExpiresAt.Equal(testTime.Add(10*time.Minute)) || p.Header.Get("Content-Type") != "image/jpeg" || len(p.Header) != 1 {
		t.Errorf("presigned = %v %v %v", p.Method, p.ExpiresAt, p.Header)
	}
	u, err := url.Parse(p.URL)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	want := map[string]string{
		"X-Amz-Algorithm":     "AWS4-HMAC-SHA256",
		"X-Amz-Expires":       "600",
		"X-Amz-SignedHeaders": "content-type;host",
		"X-Amz-Credential":    testAKID + "/20261109/local/s3/aws4_request",
		"X-Amz-Date":          "20261109T100000Z",
	}
	for k, v := range want {
		if q.Get(k) != v {
			t.Errorf("%s = %q, want %q", k, q.Get(k), v)
		}
	}
	if u.Path != testBase+"/"+testBucket+"/u/0b5c-uuid" || len(q.Get("X-Amz-Signature")) != 64 || len(q) != 6 {
		t.Errorf("presigned URL = %s", p.URL)
	}
	if strings.Contains(p.URL, testSecret) || strings.Contains(p.String(), q.Get("X-Amz-Signature")) {
		t.Error("presigned URL or String leaks the secret or signature")
	}
}

// sendPresigned sends body to a presigned request as a browser would.
func sendPresigned(t *testing.T, srv *httptest.Server, p storagex.PresignedRequest, contentType string) int {
	t.Helper()
	req, err := http.NewRequest(p.Method, p.URL, strings.NewReader("jpeg bytes"))
	if err != nil {
		t.Fatal(err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

func TestPresignPutAcceptedOnlyAsSigned(t *testing.T) {
	f, srv := newFake(t)
	c := newTestClient(t, srv)
	p, err := c.PresignPut(context.Background(), "u/0b5c-uuid", "image/jpeg", 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if code := sendPresigned(t, srv, p, "image/jpeg"); code != 200 {
		t.Fatalf("browser PUT = %d", code)
	}
	if o := f.objects["u/0b5c-uuid"]; string(o.body) != "jpeg bytes" || o.contentType != "image/jpeg" {
		t.Errorf("stored = %+v", o)
	}
	for _, ct := range []string{"text/html", ""} {
		if code := sendPresigned(t, srv, p, ct); code != 403 {
			t.Errorf("content type %q = %d, want 403", ct, code)
		}
	}
	f.now = testTime.Add(10*time.Minute + time.Second)
	if code := sendPresigned(t, srv, p, "image/jpeg"); code != 403 {
		t.Errorf("expired URL = %d, want 403", code)
	}
}

func TestPresignPutNoOverwrite(t *testing.T) {
	_, srv := newFake(t)
	guarded := newTestClient(t, srv, storagex.WithNoOverwrite())
	g, err := guarded.PresignPut(context.Background(), "u/0b5c-uuid", "image/png", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if g.Header.Get("If-None-Match") != "*" || !strings.Contains(g.URL, "X-Amz-SignedHeaders=content-type%3Bhost%3Bif-none-match") {
		t.Errorf("no-overwrite presign = %v %s", g.Header, g.URL)
	}
}

func TestPresignPutInvalid(t *testing.T) {
	_, srv := newFake(t)
	c := newTestClient(t, srv)
	ctx := context.Background()
	for _, bad := range []struct {
		ct  string
		ttl time.Duration
	}{
		{"", time.Minute}, {"image/png\n", time.Minute}, {"image/ü", time.Minute}, {strings.Repeat("a", 300), time.Minute},
		{"image/png", 0}, {"image/png", 999 * time.Millisecond}, {"image/png", 7*24*time.Hour + time.Second}, {"image/png", -time.Minute},
	} {
		_, err := c.PresignPut(ctx, "u/k", bad.ct, bad.ttl)
		if !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("PresignPut(%q, %v): err = %v", bad.ct, bad.ttl, err)
		}
		assertRedacted(t, err, testSecret, testAKID, "u/k")
	}
	if _, err := c.PresignPut(ctx, "u/k", "image/png", 7*24*time.Hour); err != nil {
		t.Errorf("7 day TTL: %v", err)
	}
}
func TestStatHeaders(t *testing.T) {
	f, srv := newFake(t)
	c := newTestClient(t, srv)
	f.override = func(w http.ResponseWriter, r *http.Request) bool {
		w.Header().Set("Content-Length", "12")
		w.Header().Set("Last-Modified", "not a date")
		w.WriteHeader(200)
		return true
	}
	info, err := c.Stat(context.Background(), "k")
	if err != nil || info.Size != 12 || !info.LastModified.IsZero() || info.ETag != "" {
		t.Errorf("Stat = %+v, %v", info, err)
	}
	f.override = func(w http.ResponseWriter, r *http.Request) bool {
		w.Header().Set("Transfer-Encoding", "chunked")
		w.WriteHeader(200)
		return true
	}
	if _, err := c.Stat(context.Background(), "k"); err == nil {
		t.Error("Stat without Content-Length: want error")
	}
}

func TestRedactedFormatting(t *testing.T) {
	cfg := Config{Endpoint: "https://proj.example/storage/v1/s3", Region: "local", Bucket: "media", AccessKeyID: testAKID, SecretAccessKey: testSecret}
	c, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var logged bytes.Buffer
	logger := newJSONLogger(&logged)
	logger.Info("config", "cfg", cfg, "client", c)
	outs := make([]string, 0, 21)
	outs = append(outs, logged.String())
	for _, verb := range []string{"%v", "%+v", "%#v", "%s", "%q"} {
		outs = append(outs, fmt.Sprintf(verb, cfg), fmt.Sprintf(verb, &cfg), fmt.Sprintf(verb, c), fmt.Sprintf(verb, *c))
	}
	for _, out := range outs {
		if strings.Contains(out, testSecret) || strings.Contains(out, testAKID) {
			t.Errorf("formatting leaks credentials: %s", out)
		}
	}
	if !strings.Contains(fmt.Sprint(c), "proj.example/storage/v1/s3/media") {
		t.Errorf("client String = %s", c)
	}
}

func TestHTTPClientIsCopiedWithoutRedirects(t *testing.T) {
	user := &http.Client{Timeout: time.Second}
	c, err := New(Config{Endpoint: "https://h.example", Region: "r", Bucket: "b", AccessKeyID: "a", SecretAccessKey: "s", HTTPClient: user})
	if err != nil {
		t.Fatal(err)
	}
	if user.CheckRedirect != nil {
		t.Error("caller's http.Client was modified")
	}
	if c.hc == user || c.hc.CheckRedirect == nil || c.hc.Timeout != time.Second {
		t.Error("client not copied with CheckRedirect")
	}
	d, err := New(Config{Endpoint: "https://h.example", Region: "r", Bucket: "b", AccessKeyID: "a", SecretAccessKey: "s"})
	if err != nil {
		t.Fatal(err)
	}
	tr, ok := d.hc.Transport.(*http.Transport)
	if !ok || tr.ResponseHeaderTimeout != 30*time.Second || tr.TLSHandshakeTimeout != 10*time.Second || d.hc.CheckRedirect == nil {
		t.Errorf("default client = %+v", d.hc)
	}
}

func newJSONLogger(w io.Writer) *slog.Logger { return slog.New(slog.NewJSONHandler(w, nil)) }

func TestDeleteTreats404AsSuccess(t *testing.T) {
	f, srv := newFake(t)
	c := newTestClient(t, srv)
	f.override = func(w http.ResponseWriter, _ *http.Request) bool {
		s3Error(w, http.StatusNotFound, "NoSuchKey")
		return true
	}
	if err := c.Delete(context.Background(), "gone"); err != nil {
		t.Errorf("Delete on 404: %v", err)
	}
}

func TestPutFromPipeIsSpooled(t *testing.T) {
	f, srv := newFake(t)
	c := newTestClient(t, srv)
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		_, _ = pw.WriteString("piped body")
		_ = pw.Close()
	}()
	defer pr.Close()
	// *os.File implements io.Seeker, but a pipe cannot seek.
	if _, err := c.Put(context.Background(), "p", "", pr); err != nil {
		t.Fatal(err)
	}
	if got := string(f.objects["p"].body); got != "piped body" {
		t.Errorf("body = %q", got)
	}
}
