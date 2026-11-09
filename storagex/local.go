package storagex

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// ErrInvalidKey is returned when a key cannot be mapped safely to an object
// name: it is empty, longer than 1024 bytes, not valid UTF-8, absolute, or it
// contains a backslash, a control character, an empty segment ("a//b", a
// trailing "/"), a "." or ".." segment, or a segment reserved for temporary
// files. The error never echoes the key.
var ErrInvalidKey = errors.New("storagex: invalid key")

// maxKeyLen matches the S3 object key limit.
const maxKeyLen = 1024

// tmpPrefix marks LocalStorage temporary files. Keys with a segment that starts
// with it are rejected, so a temporary file is never read or listed as an object.
const tmpPrefix = ".storagex-tmp-"

// validateKey reports whether key is a safe, relative, slash-separated object
// key. It returns ErrInvalidKey otherwise.
func validateKey(key string) error {
	if key == "" || len(key) > maxKeyLen || !utf8.ValidString(key) {
		return ErrInvalidKey
	}
	for _, r := range key {
		if r < 0x20 || r == 0x7f || r == '\\' {
			return ErrInvalidKey
		}
	}
	for _, seg := range strings.Split(key, "/") {
		if seg == "" || seg == "." || seg == ".." || strings.HasPrefix(seg, tmpPrefix) {
			return ErrInvalidKey
		}
	}
	// Covers platform rules on top of the checks above (Windows volume names
	// and reserved device names such as NUL or COM1).
	if !filepath.IsLocal(filepath.FromSlash(key)) {
		return ErrInvalidKey
	}
	return nil
}

// LocalStorage writes blobs to the local filesystem under a root directory.
//
// A key is a slash-separated relative path: "report.csv" is stored at
// dir/report.csv and "v/a.webp" at dir/v/a.webp, creating parent directories
// as needed. Keys that are absolute or contain "..", "\", or empty segments
// are rejected with ErrInvalidKey, and every file operation goes through an
// os.Root, so a symbolic link cannot redirect it outside dir.
//
// Put writes to a temporary file in the destination directory, syncs it, and
// renames it over the destination, so readers see either the previous object
// or the complete new one, and a failed or canceled Put leaves no partial
// object behind. Files are created with mode 0600 and directories with 0750.
type LocalStorage struct{ dir string }

// NewLocalStorage returns a LocalStorage that stores files under dir.
func NewLocalStorage(dir string) *LocalStorage { return &LocalStorage{dir: dir} }

// Put writes r to dir/<key> atomically and returns that path
// (filepath.Join(dir, key)).
func (l *LocalStorage) Put(ctx context.Context, key, _ string, r io.Reader) (string, error) {
	if err := validateKey(key); err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := os.MkdirAll(l.dir, 0o750); err != nil {
		return "", err
	}
	root, err := os.OpenRoot(l.dir)
	if err != nil {
		return "", err
	}
	defer root.Close()

	rel := filepath.FromSlash(key)
	parent := filepath.Dir(rel)
	if parent != "." {
		if err := root.MkdirAll(parent, 0o750); err != nil {
			return "", err
		}
	}
	tmp, tmpName, err := createTemp(root, parent)
	if err != nil {
		return "", err
	}
	if err := writeAndSync(ctx, tmp, r); err != nil {
		_ = root.Remove(tmpName)
		return "", err
	}
	if err := root.Rename(tmpName, rel); err != nil {
		_ = root.Remove(tmpName)
		return "", err
	}
	syncDir(root, parent)
	return filepath.Join(l.dir, rel), nil
}

// createTemp creates an exclusive temporary file in dir (relative to root) and
// returns it with its name relative to root.
func createTemp(root *os.Root, dir string) (*os.File, string, error) {
	var b [12]byte
	for range 10 {
		_, _ = rand.Read(b[:]) // crypto/rand.Read never returns an error
		name := filepath.Join(dir, tmpPrefix+hex.EncodeToString(b[:]))
		f, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return nil, "", err
		}
		return f, name, nil
	}
	return nil, "", errors.New("storagex: cannot create temporary file")
}

// writeAndSync copies r into f, flushes it to stable storage, and closes it.
// f is always closed.
func writeAndSync(ctx context.Context, f *os.File, r io.Reader) error {
	_, err := io.Copy(f, ctxReader{ctx: ctx, r: r})
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = ctx.Err()
	}
	return err
}

// syncDir flushes a directory entry change (the rename) to stable storage.
// Best effort: some platforms cannot sync a directory.
func syncDir(root *os.Root, dir string) {
	d, err := root.Open(dir)
	if err != nil {
		return
	}
	_ = d.Sync()
	_ = d.Close()
}

// ctxReader stops a copy once ctx is done.
type ctxReader struct {
	ctx context.Context
	r   io.Reader
}

func (c ctxReader) Read(p []byte) (int, error) {
	if err := c.ctx.Err(); err != nil {
		return 0, err
	}
	return c.r.Read(p)
}

// Get opens the file stored under dir/<key>. A missing key, or a key that
// names a directory, returns an error that matches fs.ErrNotExist.
func (l *LocalStorage) Get(_ context.Context, key string) (io.ReadCloser, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(l.dir)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	rel := filepath.FromSlash(key)
	f, err := root.Open(rel)
	if err != nil {
		return nil, err
	}
	fi, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	if !fi.Mode().IsRegular() {
		_ = f.Close()
		return nil, notExist("open", l.dir, key)
	}
	return f, nil
}

// Delete removes the file stored under dir/<key>. A missing key, or a key that
// names a directory, returns an error that matches fs.ErrNotExist.
func (l *LocalStorage) Delete(_ context.Context, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	root, err := os.OpenRoot(l.dir)
	if err != nil {
		return err
	}
	defer root.Close()
	rel := filepath.FromSlash(key)
	fi, err := root.Lstat(rel)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		return notExist("remove", l.dir, key)
	}
	return root.Remove(rel)
}

func notExist(op, dir, key string) error {
	return &fs.PathError{Op: op, Path: path.Join(filepath.ToSlash(dir), key), Err: fs.ErrNotExist}
}
