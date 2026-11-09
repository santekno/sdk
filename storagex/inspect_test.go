package storagex

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

type backend interface {
	Storage
	Inspector
}

var backends = map[string]func(t *testing.T, opts ...Option) backend{
	"mem":   func(_ *testing.T, opts ...Option) backend { return NewMemStorage(opts...) },
	"local": func(t *testing.T, opts ...Option) backend { return NewLocalStorage(t.TempDir(), opts...) },
}

func put(t *testing.T, s Storage, key, body string) {
	t.Helper()
	if _, err := s.Put(context.Background(), key, "text/plain", strings.NewReader(body)); err != nil {
		t.Fatalf("Put(%q): %v", key, err)
	}
}

func get(t *testing.T, s Storage, key string) string {
	t.Helper()
	r, err := s.Get(context.Background(), key)
	if err != nil {
		t.Fatalf("Get(%q): %v", key, err)
	}
	defer r.Close()
	b, _ := io.ReadAll(r)
	return string(b)
}

func keysOf(p ListPage) []string {
	out := make([]string, len(p.Objects))
	for i, o := range p.Objects {
		out[i] = o.Key
	}
	return out
}

func TestBackendsStatAndNotFound(t *testing.T) {
	for name, newB := range backends {
		t.Run(name, func(t *testing.T) {
			s := newB(t)
			ctx := context.Background()
			before := time.Now().Add(-2 * time.Second)
			put(t, s, "v/a.webp", "hello")
			info, err := s.Stat(ctx, "v/a.webp")
			if err != nil {
				t.Fatal(err)
			}
			if info.Key != "v/a.webp" || info.Size != 5 || info.LastModified.Before(before) || info.LastModified.Location() != time.UTC {
				t.Errorf("Stat = %+v", info)
			}
			if name == "mem" && (info.ContentType != "text/plain" || info.ETag == "") {
				t.Errorf("mem Stat = %+v, want content type and etag", info)
			}
			for _, op := range []func() error{
				func() error { _, err := s.Stat(ctx, "v/missing"); return err },
				func() error { _, err := s.Get(ctx, "v/missing"); return err },
				func() error { _, err := s.Stat(ctx, "v"); return err }, // a directory for local
			} {
				err := op()
				if !errors.Is(err, ErrNotFound) {
					t.Errorf("err = %v, want ErrNotFound", err)
				}
				if err != nil && strings.Contains(err.Error(), "missing") {
					t.Errorf("error echoes the key: %v", err)
				}
			}
			if _, err := s.Stat(ctx, "../x"); !errors.Is(err, ErrInvalidKey) {
				t.Errorf("Stat(../x): err = %v, want ErrInvalidKey", err)
			}
		})
	}
}

func TestBackendsList(t *testing.T) {
	for name, newB := range backends {
		t.Run(name, func(t *testing.T) {
			s := newB(t)
			ctx := context.Background()
			for _, k := range []string{"b/1", "a/3", "a0", "a/1", "a/2", "a/sub/x"} {
				put(t, s, k, k)
			}
			cases := []struct {
				opts      ListOptions
				want      []string
				wantToken string
			}{
				{ListOptions{}, []string{"a/1", "a/2", "a/3", "a/sub/x", "a0", "b/1"}, ""},
				{ListOptions{Prefix: "a/"}, []string{"a/1", "a/2", "a/3", "a/sub/x"}, ""},
				{ListOptions{Prefix: "a"}, []string{"a/1", "a/2", "a/3", "a/sub/x", "a0"}, ""},
				{ListOptions{Prefix: "a/s"}, []string{"a/sub/x"}, ""},
				{ListOptions{Prefix: "a/", Limit: 2}, []string{"a/1", "a/2"}, "a/2"},
				{ListOptions{Prefix: "a/", Limit: 2, Token: "a/2"}, []string{"a/3", "a/sub/x"}, ""},
				{ListOptions{Prefix: "a/", Limit: 4}, []string{"a/1", "a/2", "a/3", "a/sub/x"}, ""},
				{ListOptions{Prefix: "c/"}, []string{}, ""},
				{ListOptions{Prefix: "zz/deep/"}, []string{}, ""},
			}
			for _, c := range cases {
				page, err := s.List(ctx, c.opts)
				if err != nil {
					t.Fatalf("List(%+v): %v", c.opts, err)
				}
				if got := keysOf(page); fmt.Sprint(got) != fmt.Sprint(c.want) || page.NextToken != c.wantToken {
					t.Errorf("List(%+v) = %v token %q, want %v token %q", c.opts, got, page.NextToken, c.want, c.wantToken)
				}
			}
			page, _ := s.List(ctx, ListOptions{Prefix: "a/1"})
			if len(page.Objects) != 1 || page.Objects[0].Size != 3 || page.Objects[0].LastModified.IsZero() {
				t.Errorf("List object info = %+v", page.Objects)
			}
			for _, p := range []string{"/", "../", "a/../", "a//", `a\`} {
				if _, err := s.List(ctx, ListOptions{Prefix: p}); !errors.Is(err, ErrInvalidKey) {
					t.Errorf("List(prefix %q): err = %v, want ErrInvalidKey", p, err)
				}
			}
		})
	}
}

func TestBackendsNoOverwrite(t *testing.T) {
	for name, newB := range backends {
		t.Run(name, func(t *testing.T) {
			s := newB(t, WithNoOverwrite())
			ctx := context.Background()
			put(t, s, "v/hash-480.webp", "first")
			_, err := s.Put(ctx, "v/hash-480.webp", "", strings.NewReader("second"))
			if !errors.Is(err, ErrAlreadyExists) {
				t.Fatalf("second Put: err = %v, want ErrAlreadyExists", err)
			}
			if strings.Contains(err.Error(), "hash-480") {
				t.Errorf("error echoes the key: %v", err)
			}
			if got := get(t, s, "v/hash-480.webp"); got != "first" {
				t.Errorf("object = %q after refused overwrite", got)
			}
			// An existing key is refused before the body is read.
			if _, err := s.Put(ctx, "v/hash-480.webp", "", iotestErrReader{t}); !errors.Is(err, ErrAlreadyExists) {
				t.Errorf("Put existing: err = %v, want ErrAlreadyExists before reading", err)
			}
			put(t, s, "v/other.webp", "other")

			// Concurrent writers to one key: exactly one wins. Every writer
			// passes the existence pre-check before any of them finishes.
			const n = 16
			started := make(chan struct{}, n)
			release := make(chan struct{})
			go func() {
				for range n {
					<-started
				}
				close(release)
			}()
			var wg sync.WaitGroup
			var mu sync.Mutex
			wins, exists := 0, 0
			for i := range n {
				wg.Add(1)
				go func() {
					defer wg.Done()
					_, err := s.Put(ctx, "race/key", "", &barrierReader{started: started, release: release, body: fmt.Sprint(i)})
					mu.Lock()
					defer mu.Unlock()
					switch {
					case err == nil:
						wins++
					case errors.Is(err, ErrAlreadyExists):
						exists++
					default:
						t.Errorf("Put: %v", err)
					}
				}()
			}
			wg.Wait()
			if wins != 1 || exists != n-1 {
				t.Errorf("wins = %d, already exists = %d, want 1 and %d", wins, exists, n-1)
			}
		})
	}
}

func TestBackendsOverwriteByDefault(t *testing.T) {
	for name, newB := range backends {
		t.Run(name, func(t *testing.T) {
			s := newB(t)
			put(t, s, "k", "first")
			put(t, s, "k", "second")
			if got := get(t, s, "k"); got != "second" {
				t.Errorf("got %q, want overwrite", got)
			}
		})
	}
}

func TestLocalStorageNotFoundMatchesBoth(t *testing.T) {
	s := NewLocalStorage(t.TempDir())
	ctx := context.Background()
	_, gerr := s.Get(ctx, "nope")
	derr := s.Delete(ctx, "nope")
	missing := NewLocalStorage(filepath.Join(t.TempDir(), "missing"))
	_, serr := missing.Stat(ctx, "nope")
	for _, err := range []error{gerr, derr, serr} {
		if !errors.Is(err, ErrNotFound) || !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("err = %v, want ErrNotFound and fs.ErrNotExist", err)
		}
		if err != nil && strings.Contains(err.Error(), "nope") {
			t.Errorf("error echoes the key: %v", err)
		}
	}
	page, err := missing.List(ctx, ListOptions{})
	if err != nil || len(page.Objects) != 0 {
		t.Errorf("List on missing dir = %+v, %v", page, err)
	}
}

func TestLocalStorageListSkipsTempFilesAndLinks(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)
	put(t, s, "v/a.webp", "a")
	if err := os.WriteFile(filepath.Join(dir, "v", tmpPrefix+"abc"), []byte("partial"), 0o600); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		outside := t.TempDir()
		if err := os.WriteFile(filepath.Join(outside, "secret"), []byte("s"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(dir, "v", "linkdir")); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(outside, "secret"), filepath.Join(dir, "v", "linkfile")); err != nil {
			t.Fatal(err)
		}
	}
	page, err := s.List(context.Background(), ListOptions{Prefix: "v/"})
	if err != nil {
		t.Fatal(err)
	}
	if got := keysOf(page); fmt.Sprint(got) != "[v/a.webp]" {
		t.Errorf("List = %v, want only v/a.webp", got)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.List(ctx, ListOptions{}); !errors.Is(err, context.Canceled) {
		t.Errorf("List with canceled ctx: err = %v", err)
	}
}

func TestLocalStorageNoOverwriteErrors(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir, WithNoOverwrite())
	ctx := context.Background()
	if _, err := s.Put(ctx, "f", "", &failingReader{}); !errors.Is(err, errBrokenUpload) {
		t.Errorf("Put: err = %v, want reader error", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "f")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("object exists after failed Put: %v", err)
	}
	put(t, s, "d/x", "x")
	if _, err := s.Put(ctx, "d/x/y", "", strings.NewReader("y")); err == nil || errors.Is(err, ErrAlreadyExists) {
		t.Errorf("Put under a file: err = %v, want a filesystem error", err)
	}
	if tmp := tempFiles(t, dir); len(tmp) != 0 {
		t.Errorf("temporary files left: %v", tmp)
	}
}

func TestMemStorageErrors(t *testing.T) {
	s := NewMemStorage()
	ctx := context.Background()
	if _, err := s.Put(ctx, "a", "", &failingReader{}); !errors.Is(err, errBrokenUpload) {
		t.Errorf("Put: err = %v", err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := s.Put(canceled, "a", "", strings.NewReader("x")); !errors.Is(err, context.Canceled) {
		t.Errorf("Put canceled: err = %v", err)
	}
	for _, key := range []string{"", "../a", "/a", "a//b"} {
		if _, err := s.Put(ctx, key, "", strings.NewReader("x")); !errors.Is(err, ErrInvalidKey) {
			t.Errorf("Put(%q): err = %v", key, err)
		}
		if _, err := s.Get(ctx, key); !errors.Is(err, ErrInvalidKey) {
			t.Errorf("Get(%q): err = %v", key, err)
		}
		if err := s.Delete(ctx, key); !errors.Is(err, ErrInvalidKey) {
			t.Errorf("Delete(%q): err = %v", key, err)
		}
	}
	if err := s.Delete(ctx, "missing"); err != nil {
		t.Errorf("Delete missing: %v", err)
	}
}

func TestValidateKeyAndPrefix(t *testing.T) {
	good := []string{"a", "v/a.webp", ".hidden", "a..b", "x/.y/z", "ünïcode", "u/0b5c1f0e-uuid"}
	bad := []string{"", "/a", "a/", "a//b", ".", "..", "a/../b", "./a", `a\b`, "a\x00", "a\tb", "\xff", tmpPrefix + "1", "a/" + tmpPrefix}
	for _, k := range good {
		if err := ValidateKey(k); err != nil {
			t.Errorf("ValidateKey(%q) = %v", k, err)
		}
		if err := ValidatePrefix(k); err != nil {
			t.Errorf("ValidatePrefix(%q) = %v", k, err)
		}
	}
	for _, k := range bad {
		if err := ValidateKey(k); !errors.Is(err, ErrInvalidKey) {
			t.Errorf("ValidateKey(%q) = %v, want ErrInvalidKey", k, err)
		}
	}
	for _, p := range []string{"", "a/", "v/", "a/b/", "a/b"} {
		if err := ValidatePrefix(p); err != nil {
			t.Errorf("ValidatePrefix(%q) = %v", p, err)
		}
	}
	for _, p := range []string{"/", "//", "a//", "../", "a/./", `a\`, strings.Repeat("p", MaxKeyLen+1)} {
		if err := ValidatePrefix(p); !errors.Is(err, ErrInvalidKey) {
			t.Errorf("ValidatePrefix(%q) = %v, want ErrInvalidKey", p, err)
		}
	}
}

func TestListOptionsPageLimit(t *testing.T) {
	for in, want := range map[int]int{-1: MaxListLimit, 0: MaxListLimit, 1: 1, 999: 999, MaxListLimit: MaxListLimit, MaxListLimit + 1: MaxListLimit} {
		if got := (ListOptions{Limit: in}).PageLimit(); got != want {
			t.Errorf("PageLimit(%d) = %d, want %d", in, got, want)
		}
	}
}

func TestNewOptions(t *testing.T) {
	if o := NewOptions(); o.NoOverwrite {
		t.Error("default NoOverwrite = true")
	}
	if o := NewOptions(nil, WithNoOverwrite()); !o.NoOverwrite {
		t.Error("WithNoOverwrite not applied")
	}
}

const testSignature = "X-Amz-Signature=deadbeefcafe0123456789"

func TestPresignedRequestRedaction(t *testing.T) {
	p := PresignedRequest{
		Method:    http.MethodPut,
		URL:       "https://user:pw@example.test/storage/v1/s3/media-uploads/u/1?X-Amz-Credential=AKIDSECRET%2F20261109&" + testSignature + "#frag",
		Header:    http.Header{"Content-Type": {"image/jpeg"}},
		ExpiresAt: time.Date(2026, 11, 9, 10, 10, 0, 0, time.UTC),
	}
	var logged bytes.Buffer
	slog.New(slog.NewJSONHandler(&logged, nil)).Info("presigned", "req", p, "ptr", &p)
	outputs := make([]string, 0, 13)
	outputs = append(outputs, p.String(), p.GoString(), logged.String())
	for _, verb := range []string{"%v", "%+v", "%#v", "%s", "%q"} {
		outputs = append(outputs, fmt.Sprintf(verb, p), fmt.Sprintf(verb, &p))
	}
	for _, out := range outputs {
		for _, secret := range []string{"deadbeefcafe", "AKIDSECRET", "user:pw", "frag"} {
			if strings.Contains(out, secret) {
				t.Errorf("output leaks %q: %s", secret, out)
			}
		}
	}
	if want := "PUT https://example.test/storage/v1/s3/media-uploads/u/1?REDACTED (expires 2026-11-09T10:10:00Z)"; p.String() != want {
		t.Errorf("String = %q, want %q", p.String(), want)
	}
	if !strings.Contains(logged.String(), `"url":"https://example.test/storage/v1/s3/media-uploads/u/1?REDACTED"`) {
		t.Errorf("slog output = %s", logged.String())
	}
}

func TestRedactURL(t *testing.T) {
	cases := map[string]string{
		"https://h/p":            "https://h/p",
		"https://h/p?":           "https://h/p?REDACTED",
		"https://h/p?a=b":        "https://h/p?REDACTED",
		"http://u:p@h:9000/b/k":  "http://h:9000/b/k",
		"https://h/p#x":          "https://h/p",
		"://bad":                 "REDACTED",
		"https://h/a%20b?sig=1c": "https://h/a%20b?REDACTED",
	}
	for in, want := range cases {
		if got := RedactURL(in); got != want {
			t.Errorf("RedactURL(%q) = %q, want %q", in, got, want)
		}
	}
}

// iotestErrReader fails the test when read.
type iotestErrReader struct{ t *testing.T }

func (r iotestErrReader) Read([]byte) (int, error) {
	r.t.Error("body was read")
	return 0, io.ErrUnexpectedEOF
}

func TestLocalStorageStatAndListStayInsideDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	dir := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret"), []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}
	s := NewLocalStorage(dir)
	if info, err := s.Stat(context.Background(), "link/secret"); err == nil {
		t.Errorf("Stat through symlink = %+v, want error", info)
	}
	page, err := s.List(context.Background(), ListOptions{Prefix: "link/"})
	if err == nil && len(page.Objects) != 0 {
		t.Errorf("List through symlink = %v", keysOf(page))
	}
}

// barrierReader signals started on its first Read and then waits for release.
type barrierReader struct {
	started chan<- struct{}
	release <-chan struct{}
	body    string
	once    sync.Once
	r       io.Reader
}

func (b *barrierReader) Read(p []byte) (int, error) {
	b.once.Do(func() {
		b.started <- struct{}{}
		<-b.release
		b.r = strings.NewReader(b.body)
	})
	return b.r.Read(p)
}
