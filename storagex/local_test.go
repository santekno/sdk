package storagex

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
)

func readKey(t *testing.T, s *LocalStorage, key string) string {
	t.Helper()
	r, err := s.Get(context.Background(), key)
	if err != nil {
		t.Fatalf("Get(%q): %v", key, err)
	}
	defer r.Close()
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read %q: %v", key, err)
	}
	return string(b)
}

// tempFiles returns every file under dir whose name marks a temporary file.
func tempFiles(t *testing.T, dir string) []string {
	t.Helper()
	var found []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.HasPrefix(d.Name(), tmpPrefix) {
			found = append(found, p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return found
}

// Flat keys keep the v0.5.0 behavior: dir/<key>, returned path, Get, Delete.
func TestLocalStorageFlatKeyUnchanged(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "created", "on", "put")
	s := NewLocalStorage(dir)
	ctx := context.Background()

	got, err := s.Put(ctx, "report.csv", "text/csv", strings.NewReader("x,y,z"))
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "report.csv"); got != want {
		t.Errorf("Put path = %q, want %q", got, want)
	}
	b, err := os.ReadFile(filepath.Join(dir, "report.csv"))
	if err != nil || string(b) != "x,y,z" {
		t.Fatalf("file content = %q, %v", b, err)
	}
	if _, err := s.Put(ctx, "report.csv", "text/csv", strings.NewReader("new")); err != nil {
		t.Fatal(err)
	}
	if got := readKey(t, s, "report.csv"); got != "new" {
		t.Errorf("after overwrite got %q, want %q", got, "new")
	}
	if err := s.Delete(ctx, "report.csv"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(ctx, "report.csv"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Get after Delete: err = %v, want fs.ErrNotExist", err)
	}
	if err := s.Delete(ctx, "report.csv"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Delete missing: err = %v, want fs.ErrNotExist", err)
	}
}

func TestLocalStorageKeepsKeyPrefix(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)
	ctx := context.Background()

	pv, err := s.Put(ctx, "v/a.webp", "image/webp", strings.NewReader("variant"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Put(ctx, "u/a.webp", "image/webp", strings.NewReader("upload")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Put(ctx, "v/deep/er/b.webp", "image/webp", strings.NewReader("deep")); err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "v", "a.webp"); pv != want {
		t.Errorf("Put path = %q, want %q", pv, want)
	}
	for key, want := range map[string]string{"v/a.webp": "variant", "u/a.webp": "upload", "v/deep/er/b.webp": "deep"} {
		if got := readKey(t, s, key); got != want {
			t.Errorf("Get(%q) = %q, want %q", key, got, want)
		}
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(key))); err != nil {
			t.Errorf("file for %q: %v", key, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "a.webp")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("basename collision file exists: %v", err)
	}
	if err := s.Delete(ctx, "v/a.webp"); err != nil {
		t.Fatal(err)
	}
	if got := readKey(t, s, "u/a.webp"); got != "upload" {
		t.Errorf("Delete(v/a.webp) touched u/a.webp: %q", got)
	}
}

func TestLocalStorageRejectsInvalidKeys(t *testing.T) {
	keys := []string{
		"",
		"/abs",
		"/etc/passwd",
		"../escape",
		"a/../../escape",
		"a/../b",
		"..",
		".",
		"./a",
		"a/./b",
		"a//b",
		"a/",
		"/",
		`a\b`,
		`..\escape`,
		"a\x00b",
		"a\nb",
		"a\x7fb",
		"\xff\xfe",
		tmpPrefix + "x",
		"v/" + tmpPrefix + "x",
		strings.Repeat("k", MaxKeyLen+1),
	}
	parent := t.TempDir()
	dir := filepath.Join(parent, "root")
	s := NewLocalStorage(dir)
	ctx := context.Background()
	for _, key := range keys {
		_, err := s.Put(ctx, key, "", strings.NewReader("pwned"))
		if !errors.Is(err, ErrInvalidKey) {
			t.Errorf("Put(%q): err = %v, want ErrInvalidKey", key, err)
		}
		if err != nil && key != "" && strings.Contains(err.Error(), key) {
			t.Errorf("Put(%q): error echoes the key: %v", key, err)
		}
		if r, err := s.Get(ctx, key); !errors.Is(err, ErrInvalidKey) {
			if r != nil {
				r.Close()
			}
			t.Errorf("Get(%q): err = %v, want ErrInvalidKey", key, err)
		}
		if err := s.Delete(ctx, key); !errors.Is(err, ErrInvalidKey) {
			t.Errorf("Delete(%q): err = %v, want ErrInvalidKey", key, err)
		}
	}
	if _, err := os.Stat(filepath.Join(parent, "escape")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a file was written outside dir: %v", err)
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		t.Errorf("rejected key left %q in dir", e.Name())
	}
}

func TestValidateKeyLength(t *testing.T) {
	seg := strings.Repeat("k", 127) + "/"
	key := strings.Repeat(seg, MaxKeyLen/len(seg)-1)
	key += strings.Repeat("x", MaxKeyLen-len(key))
	if len(key) != MaxKeyLen {
		t.Fatalf("test key length = %d", len(key))
	}
	if err := ValidateKey(key); err != nil {
		t.Errorf("key of %d bytes: %v", len(key), err)
	}
	if err := ValidateKey(key + "x"); !errors.Is(err, ErrInvalidKey) {
		t.Errorf("key of %d bytes: err = %v, want ErrInvalidKey", len(key)+1, err)
	}
}

func TestLocalStorageAcceptsValidKeys(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)
	ctx := context.Background()
	for _, key := range []string{"a", ".hidden", "a..b", "x/.y/z", "ünïcode/ok.txt", "d-1_2.3/e~f", strings.Repeat("k", 200) + "/" + strings.Repeat("j", 200)} {
		if _, err := s.Put(ctx, key, "", strings.NewReader(key)); err != nil {
			t.Errorf("Put(%q): %v", key, err)
			continue
		}
		if got := readKey(t, s, key); got != key {
			t.Errorf("Get(%q) = %q", key, got)
		}
	}
}

// failingReader returns some data and then an error, like a dropped upload.
type failingReader struct{ sent bool }

var errBrokenUpload = errors.New("upload interrupted")

func (f *failingReader) Read(p []byte) (int, error) {
	if f.sent {
		return 0, errBrokenUpload
	}
	f.sent = true
	return copy(p, "partial"), nil
}

func TestLocalStorageFailedCopyLeavesNoObject(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)
	ctx := context.Background()

	if _, err := s.Put(ctx, "v/new.webp", "", &failingReader{}); !errors.Is(err, errBrokenUpload) {
		t.Fatalf("Put: err = %v, want the reader error", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "v", "new.webp")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("partial object exists at the key: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "new.webp")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("partial object exists at the basename: %v", err)
	}
	if _, err := s.Get(ctx, "v/new.webp"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Get after failed Put: err = %v, want fs.ErrNotExist", err)
	}
	if tmp := tempFiles(t, dir); len(tmp) != 0 {
		t.Errorf("temporary files left: %v", tmp)
	}
}

func TestLocalStorageFailedOverwriteKeepsPrevious(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)
	ctx := context.Background()

	if _, err := s.Put(ctx, "backup.dump", "", strings.NewReader("complete previous object")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Put(ctx, "backup.dump", "", &failingReader{}); !errors.Is(err, errBrokenUpload) {
		t.Fatalf("Put: err = %v, want the reader error", err)
	}
	if got := readKey(t, s, "backup.dump"); got != "complete previous object" {
		t.Errorf("previous object = %q after failed overwrite", got)
	}
	if tmp := tempFiles(t, dir); len(tmp) != 0 {
		t.Errorf("temporary files left: %v", tmp)
	}
}

// cancelingReader cancels the context after the first chunk.
type cancelingReader struct {
	cancel context.CancelFunc
	sent   bool
}

func (c *cancelingReader) Read(p []byte) (int, error) {
	if c.sent {
		return copy(p, "rest"), io.EOF
	}
	c.sent = true
	c.cancel()
	return copy(p, "first"), nil
}

func TestLocalStorageCanceledContext(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Put(ctx, "before.txt", "", strings.NewReader("data")); !errors.Is(err, context.Canceled) {
		t.Errorf("Put with canceled ctx: err = %v, want context.Canceled", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "before.txt")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("object written despite canceled ctx: %v", err)
	}

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	if _, err := s.Put(ctx, "during.txt", "", &cancelingReader{cancel: cancel}); !errors.Is(err, context.Canceled) {
		t.Errorf("Put canceled mid-copy: err = %v, want context.Canceled", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "during.txt")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("object written after cancel: %v", err)
	}
	if tmp := tempFiles(t, dir); len(tmp) != 0 {
		t.Errorf("temporary files left: %v", tmp)
	}
}

func TestLocalStorageDirectoryIsNotAnObject(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)
	ctx := context.Background()
	if _, err := s.Put(ctx, "v/a.webp", "", strings.NewReader("a")); err != nil {
		t.Fatal(err)
	}
	if r, err := s.Get(ctx, "v"); !errors.Is(err, fs.ErrNotExist) {
		if r != nil {
			r.Close()
		}
		t.Errorf("Get(directory): err = %v, want fs.ErrNotExist", err)
	}
	if err := s.Delete(ctx, "v"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Delete(directory): err = %v, want fs.ErrNotExist", err)
	}
	if got := readKey(t, s, "v/a.webp"); got != "a" {
		t.Errorf("object under directory = %q", got)
	}
	// A key whose parent is an existing object cannot be created.
	if _, err := s.Put(ctx, "v/a.webp/child", "", strings.NewReader("c")); err == nil {
		t.Error("Put under a file: want error")
	}
	// A key that names an existing non-empty directory cannot be overwritten.
	if _, err := s.Put(ctx, "v", "", strings.NewReader("c")); err == nil {
		t.Error("Put over a directory: want error")
	}
	if tmp := tempFiles(t, dir); len(tmp) != 0 {
		t.Errorf("temporary files left: %v", tmp)
	}
}

func TestLocalStorageSymlinkCannotEscape(t *testing.T) {
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
	ctx := context.Background()

	if _, err := s.Put(ctx, "link/x", "", strings.NewReader("pwned")); err == nil {
		t.Error("Put through symlink: want error")
	}
	if _, err := os.Stat(filepath.Join(outside, "x")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("file written outside dir through symlink: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "x")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("file written at basename: %v", err)
	}
	if r, err := s.Get(ctx, "link/secret"); err == nil {
		r.Close()
		t.Error("Get through symlink: want error")
	}
	if err := s.Delete(ctx, "link/secret"); err == nil {
		t.Error("Delete through symlink: want error")
	}
	if _, err := os.Stat(filepath.Join(outside, "secret")); err != nil {
		t.Errorf("file outside dir was removed: %v", err)
	}
}

func TestLocalStorageFileMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX modes only")
	}
	dir := t.TempDir()
	s := NewLocalStorage(dir)
	p, err := s.Put(context.Background(), "m/f.bin", "", strings.NewReader("x"))
	if err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != 0o600 {
		t.Errorf("file mode = %o, want 600", got)
	}
}

func TestLocalStorageRootErrors(t *testing.T) {
	ctx := context.Background()
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := NewLocalStorage(file) // dir is a regular file
	if _, err := s.Put(ctx, "a", "", strings.NewReader("x")); err == nil {
		t.Error("Put with a file as dir: want error")
	}
	missing := NewLocalStorage(filepath.Join(t.TempDir(), "missing"))
	if _, err := missing.Get(ctx, "a"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Get with missing dir: err = %v, want fs.ErrNotExist", err)
	}
	if err := missing.Delete(ctx, "a"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Delete with missing dir: err = %v, want fs.ErrNotExist", err)
	}
}

const keySentinel = "SENTINELKEY9f3a"

// assertNoKey checks the rendered error never contains the key.
func assertNoKey(t *testing.T, what string, err error) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: want an error", what)
		return
	}
	for _, verb := range []string{"%v", "%+v", "%#v", "%s", "%q"} {
		if out := fmt.Sprintf(verb, err); strings.Contains(out, keySentinel) {
			t.Errorf("%s: %s echoes the key: %s", what, verb, out)
		}
	}
}

func TestLocalStorageErrorsDoNotEchoKeys(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)
	ctx := context.Background()
	if _, err := s.Put(ctx, "file", "", strings.NewReader("f")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Put(ctx, keySentinel+"/dir/obj", "", strings.NewReader("o")); err != nil {
		t.Fatal(err)
	}

	_, err := s.Get(ctx, keySentinel+"/missing")
	assertNoKey(t, "Get missing", err)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Get missing: err = %v, want fs.ErrNotExist", err)
	}
	err = s.Delete(ctx, keySentinel+"/missing")
	assertNoKey(t, "Delete missing", err)
	_, err = s.Get(ctx, keySentinel+"/dir")
	assertNoKey(t, "Get directory", err)
	err = s.Delete(ctx, keySentinel+"/dir")
	assertNoKey(t, "Delete directory", err)
	_, err = s.Put(ctx, "file/"+keySentinel, "", strings.NewReader("x"))
	assertNoKey(t, "Put under a file", err)
	var pe *fs.PathError
	if !errors.As(err, &pe) || errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Put under a file: err = %v, want an unwrappable *fs.PathError, not ErrNotExist", err)
	}
	// Get under a file (ENOTDIR) is not found but still unwraps to the errno.
	_, err = s.Get(ctx, "file/"+keySentinel)
	assertNoKey(t, "Get under a file", err)
	if !errors.Is(err, fs.ErrNotExist) || !errors.Is(err, syscall.ENOTDIR) {
		t.Errorf("Get under a file: err = %v, want fs.ErrNotExist wrapping ENOTDIR", err)
	}
	_, err = s.Put(ctx, keySentinel+"/dir", "", strings.NewReader("x"))
	assertNoKey(t, "Put over a directory", err)
	var le *os.LinkError
	if !errors.As(err, &le) {
		t.Errorf("Put over a directory: err = %v, want an unwrappable *os.LinkError", err)
	}
	if !strings.HasPrefix(err.Error(), "storagex: local put: ") {
		t.Errorf("message = %q", err)
	}

	if runtime.GOOS != "windows" {
		if err := os.Symlink(t.TempDir(), filepath.Join(dir, "link")); err != nil {
			t.Fatal(err)
		}
		_, err = s.Put(ctx, "link/"+keySentinel, "", strings.NewReader("x"))
		assertNoKey(t, "Put through symlink", err)
		_, err = s.Get(ctx, "link/"+keySentinel)
		assertNoKey(t, "Get through symlink", err)
	}
	if _, err := s.Get(ctx, "file"); err != nil {
		t.Errorf("unrelated key broken: %v", err)
	}
}

// A path through an existing file (ENOTDIR) is a missing object.
func TestLocalStorageNotADirectoryIsNotFound(t *testing.T) {
	s := NewLocalStorage(t.TempDir())
	ctx := context.Background()
	if _, err := s.Put(ctx, "file", "", strings.NewReader("f")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(ctx, "file/child"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Get under a file: err = %v, want fs.ErrNotExist", err)
	}
	if err := s.Delete(ctx, "file/child"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Delete under a file: err = %v, want fs.ErrNotExist", err)
	}
}
