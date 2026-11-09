package s3

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/santekno/sdk/storagex"
)

// itEnv is the state shared by the integration subtests.
type itEnv struct {
	cfg     Config
	c       *Client // overwrites
	guarded *Client // storagex.WithNoOverwrite
	ctx     context.Context
	prefix  string
	written []string
}

// TestIntegration runs against a real S3-compatible server when
// STORAGEX_S3_IT_ENDPOINT is set, e.g. the Supabase CLI:
//
//	STORAGEX_S3_IT_ENDPOINT=http://127.0.0.1:54321/storage/v1/s3 \
//	STORAGEX_S3_IT_REGION=local \
//	STORAGEX_S3_IT_BUCKET=<existing bucket> \
//	STORAGEX_S3_IT_ACCESS_KEY_ID=... STORAGEX_S3_IT_SECRET_ACCESS_KEY=... \
//	go test ./storagex/s3/ -run TestIntegration -v
//
// It only writes under a random "it/<hex>/" prefix and deletes what it wrote.
// Set STORAGEX_S3_IT_SKIP_NO_OVERWRITE=1 for servers that ignore
// If-None-Match on PUT.
func TestIntegration(t *testing.T) {
	endpoint := os.Getenv("STORAGEX_S3_IT_ENDPOINT")
	if endpoint == "" {
		t.Skip("STORAGEX_S3_IT_ENDPOINT not set")
	}
	env := &itEnv{cfg: Config{
		Endpoint:        endpoint,
		Region:          os.Getenv("STORAGEX_S3_IT_REGION"),
		Bucket:          os.Getenv("STORAGEX_S3_IT_BUCKET"),
		AccessKeyID:     os.Getenv("STORAGEX_S3_IT_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("STORAGEX_S3_IT_SECRET_ACCESS_KEY"),
	}}
	var err error
	if env.c, err = New(env.cfg); err != nil {
		t.Fatal(err)
	}
	if env.guarded, err = New(env.cfg, storagex.WithNoOverwrite()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	env.ctx = ctx
	var rnd [6]byte
	_, _ = rand.Read(rnd[:])
	env.prefix = "it/" + hex.EncodeToString(rnd[:]) + "/"
	t.Cleanup(func() { env.cleanup(t) })

	t.Run("put get stat seekable and spooled", env.putGetStat)
	t.Run("not found", env.notFound)
	t.Run("list pages", env.listPages)
	t.Run("no overwrite", env.noOverwrite)
	t.Run("presigned put", env.presignedPut)
	t.Run("wrong secret", env.wrongSecret)
}

func (e *itEnv) cleanup(t *testing.T) {
	for _, k := range e.written {
		if err := e.c.Delete(context.Background(), k); err != nil {
			t.Errorf("cleanup: %v", err)
		}
	}
}

func (e *itEnv) put(t *testing.T, cl *Client, key, ct string, r io.Reader) {
	t.Helper()
	if _, err := cl.Put(e.ctx, e.prefix+key, ct, r); err != nil {
		t.Fatalf("Put(%s): %v", key, err)
	}
	e.written = append(e.written, e.prefix+key)
}

func (e *itEnv) read(t *testing.T, key string) string {
	t.Helper()
	rc, err := e.c.Get(e.ctx, e.prefix+key)
	if err != nil {
		t.Fatalf("Get(%s): %v", key, err)
	}
	defer rc.Close()
	b, _ := io.ReadAll(rc)
	return string(b)
}

func (e *itEnv) putGetStat(t *testing.T) {
	e.put(t, e.c, "a b+ü.webp", "image/webp", bytes.NewReader([]byte("seekable")))
	e.put(t, e.c, "dir/spooled.bin", "application/octet-stream", onlyReader{strings.NewReader("spooled body")})
	if got := e.read(t, "a b+ü.webp"); got != "seekable" {
		t.Errorf("Get = %q", got)
	}
	info, err := e.c.Stat(e.ctx, e.prefix+"dir/spooled.bin")
	if err != nil {
		t.Fatal(err)
	}
	if info.Size != int64(len("spooled body")) || info.LastModified.IsZero() {
		t.Errorf("Stat = %+v", info)
	}
	t.Logf("Stat content type %q etag %q", info.ContentType, info.ETag)
}

func (e *itEnv) notFound(t *testing.T) {
	if _, err := e.c.Stat(e.ctx, e.prefix+"missing"); !errors.Is(err, storagex.ErrNotFound) {
		t.Errorf("Stat missing: %v", err)
	}
	if _, err := e.c.Get(e.ctx, e.prefix+"missing"); !errors.Is(err, storagex.ErrNotFound) {
		t.Errorf("Get missing: %v", err)
	}
	if err := e.c.Delete(e.ctx, e.prefix+"missing"); err != nil {
		t.Errorf("Delete missing: %v", err)
	}
}

func (e *itEnv) listPages(t *testing.T) {
	for _, k := range []string{"l/1", "l/2", "l/3"} {
		e.put(t, e.c, k, "text/plain", strings.NewReader(k))
	}
	var got []string
	opts := storagex.ListOptions{Prefix: e.prefix + "l/", Limit: 2}
	for range 5 {
		page, err := e.c.List(e.ctx, opts)
		if err != nil {
			t.Fatal(err)
		}
		for _, o := range page.Objects {
			got = append(got, strings.TrimPrefix(o.Key, e.prefix))
		}
		if page.NextToken == "" {
			break
		}
		opts.Token = page.NextToken
	}
	if strings.Join(got, ",") != "l/1,l/2,l/3" {
		t.Errorf("List = %v", got)
	}
}

func (e *itEnv) noOverwrite(t *testing.T) {
	if os.Getenv("STORAGEX_S3_IT_SKIP_NO_OVERWRITE") != "" {
		t.Skip("server ignores If-None-Match")
	}
	e.put(t, e.guarded, "once", "text/plain", strings.NewReader("first"))
	_, err := e.guarded.Put(e.ctx, e.prefix+"once", "text/plain", strings.NewReader("second"))
	if !errors.Is(err, storagex.ErrAlreadyExists) {
		t.Errorf("second Put: %v, want ErrAlreadyExists", err)
	}
	if got := e.read(t, "once"); got != "first" {
		t.Errorf("object = %q after refused overwrite", got)
	}
}

func (e *itEnv) presignedPut(t *testing.T) {
	key := e.prefix + "presigned.jpg"
	p, err := e.c.PresignPut(e.ctx, key, "image/jpeg", 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	send := func(ct string) int {
		req, err := http.NewRequestWithContext(e.ctx, p.Method, p.URL, strings.NewReader("jpeg"))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", ct)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	if code := send("text/html"); code < 400 {
		t.Errorf("presigned PUT with another content type = %d, want 4xx", code)
	}
	if code := send("image/jpeg"); code != 200 {
		t.Fatalf("presigned PUT = %d", code)
	}
	e.written = append(e.written, key)
	info, err := e.c.Stat(e.ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size != 4 || info.ContentType != "image/jpeg" {
		t.Errorf("Stat = %+v", info)
	}
}

func (e *itEnv) wrongSecret(t *testing.T) {
	bad := e.cfg
	bad.SecretAccessKey += "x"
	bc, err := New(bad)
	if err != nil {
		t.Fatal(err)
	}
	_, err = bc.Stat(e.ctx, e.prefix+"dir/spooled.bin")
	var se *Error
	if !errors.As(err, &se) || se.StatusCode != 403 {
		t.Errorf("wrong secret: %v", err)
	}
	assertRedacted(t, err, e.cfg.SecretAccessKey, e.cfg.AccessKeyID, e.prefix)
}
