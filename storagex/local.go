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
	"sort"
	"strings"
)

// localKey validates key and maps it to a path relative to the root. On top
// of ValidateKey it applies the platform rules of filepath.IsLocal (Windows
// volume names and reserved device names such as NUL or COM1).
func localKey(key string) (string, error) {
	if err := ValidateKey(key); err != nil {
		return "", err
	}
	rel := filepath.FromSlash(key)
	if !filepath.IsLocal(rel) {
		return "", ErrInvalidKey
	}
	return rel, nil
}

// LocalStorage writes blobs to the local filesystem under a root directory.
//
// A key is a slash-separated relative path: "report.csv" is stored at
// dir/report.csv and "v/a.webp" at dir/v/a.webp, creating parent directories
// as needed. Keys rejected by ValidateKey (absolute, "..", "\", empty
// segments, ...) return ErrInvalidKey, and every file operation goes through
// an os.Root, so a symbolic link cannot redirect it outside dir.
//
// Put writes to a temporary file in the destination directory, syncs it, and
// renames it over the destination (or hard-links it, with WithNoOverwrite),
// so readers see either the previous object or the complete new one, and a
// failed or canceled Put leaves no partial object behind. Files are created
// with mode 0600 and directories with 0750.
//
// Get and Delete of a missing key return an error matching both ErrNotFound
// and fs.ErrNotExist. Stat and List report no ContentType or ETag.
type LocalStorage struct {
	dir  string
	opts Options
}

// NewLocalStorage returns a LocalStorage that stores files under dir.
func NewLocalStorage(dir string, opts ...Option) *LocalStorage {
	return &LocalStorage{dir: dir, opts: NewOptions(opts...)}
}

// Put writes r to dir/<key> atomically and returns that path
// (filepath.Join(dir, key)).
func (l *LocalStorage) Put(ctx context.Context, key, _ string, r io.Reader) (string, error) {
	rel, err := localKey(key)
	if err != nil {
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

	if l.opts.NoOverwrite {
		if _, err := root.Lstat(rel); err == nil {
			return "", ErrAlreadyExists
		}
	}
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
	if l.opts.NoOverwrite {
		// link fails if the destination exists, so a concurrent writer that
		// got there first is never replaced.
		err = root.Link(tmpName, rel)
		_ = root.Remove(tmpName)
		if errors.Is(err, fs.ErrExist) {
			return "", ErrAlreadyExists
		}
	} else {
		err = root.Rename(tmpName, rel)
		if err != nil {
			_ = root.Remove(tmpName)
		}
	}
	if err != nil {
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

// localNotFound matches ErrNotFound and, through Unwrap, fs.ErrNotExist (the
// v0.5.0 error). Its message omits the path.
type localNotFound struct{ cause error }

func (e *localNotFound) Error() string        { return ErrNotFound.Error() }
func (e *localNotFound) Is(target error) bool { return target == ErrNotFound }
func (e *localNotFound) Unwrap() error        { return e.cause }

// notFound wraps err when it reports a missing file and returns it unchanged
// otherwise.
func notFound(err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return &localNotFound{cause: err}
	}
	return err
}

func dirNotFound(op, dir, key string) error {
	return &localNotFound{cause: &fs.PathError{Op: op, Path: path.Join(filepath.ToSlash(dir), key), Err: fs.ErrNotExist}}
}

// Get opens the file stored under dir/<key>. A missing key, or a key that
// names a directory, returns an error matching ErrNotFound and fs.ErrNotExist.
func (l *LocalStorage) Get(_ context.Context, key string) (io.ReadCloser, error) {
	rel, err := localKey(key)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(l.dir)
	if err != nil {
		return nil, notFound(err)
	}
	defer root.Close()
	f, err := root.Open(rel)
	if err != nil {
		return nil, notFound(err)
	}
	fi, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	if !fi.Mode().IsRegular() {
		_ = f.Close()
		return nil, dirNotFound("open", l.dir, key)
	}
	return f, nil
}

// Delete removes the file stored under dir/<key>. A missing key, or a key that
// names a directory, returns an error matching ErrNotFound and fs.ErrNotExist.
func (l *LocalStorage) Delete(_ context.Context, key string) error {
	rel, err := localKey(key)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(l.dir)
	if err != nil {
		return notFound(err)
	}
	defer root.Close()
	fi, err := root.Lstat(rel)
	if err != nil {
		return notFound(err)
	}
	if fi.IsDir() {
		return dirNotFound("remove", l.dir, key)
	}
	return notFound(root.Remove(rel))
}

// Stat describes the file stored under dir/<key>. A missing key, or a key that
// names a directory, returns an error matching ErrNotFound.
func (l *LocalStorage) Stat(_ context.Context, key string) (ObjectInfo, error) {
	rel, err := localKey(key)
	if err != nil {
		return ObjectInfo{}, err
	}
	root, err := os.OpenRoot(l.dir)
	if err != nil {
		return ObjectInfo{}, notFound(err)
	}
	defer root.Close()
	fi, err := root.Stat(rel)
	if err != nil {
		return ObjectInfo{}, notFound(err)
	}
	if !fi.Mode().IsRegular() {
		return ObjectInfo{}, dirNotFound("stat", l.dir, key)
	}
	return ObjectInfo{Key: key, Size: fi.Size(), LastModified: fi.ModTime().UTC()}, nil
}

// List returns one page of regular files whose key starts with opts.Prefix,
// sorted by key. It walks the directory named by the prefix up to its last
// "/", skips symbolic links and temporary files, and does not follow links.
// Every call walks that whole subtree, so it suits small trees (tests, local
// development, backups), not millions of objects. NextToken is the last key
// of the page.
func (l *LocalStorage) List(ctx context.Context, opts ListOptions) (ListPage, error) {
	if err := ValidatePrefix(opts.Prefix); err != nil {
		return ListPage{}, err
	}
	root, err := os.OpenRoot(l.dir)
	if errors.Is(err, fs.ErrNotExist) {
		return ListPage{}, nil
	}
	if err != nil {
		return ListPage{}, err
	}
	defer root.Close()

	infos, err := walkObjects(ctx, root, opts)
	if err != nil {
		return ListPage{}, err
	}
	sort.Slice(infos, func(i, j int) bool { return infos[i].Key < infos[j].Key })
	var page ListPage
	if limit := opts.PageLimit(); len(infos) > limit {
		infos = infos[:limit]
		page.NextToken = infos[limit-1].Key
	}
	page.Objects = infos
	return page, nil
}

// walkObjects collects the regular files under root that match opts, walking
// only the directory named by the prefix up to its last "/".
func walkObjects(ctx context.Context, root *os.Root, opts ListOptions) ([]ObjectInfo, error) {
	start := "."
	if i := strings.LastIndex(opts.Prefix, "/"); i >= 0 {
		start = opts.Prefix[:i]
	}
	var infos []ObjectInfo
	err := fs.WalkDir(root.FS(), start, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if p == start && errors.Is(err, fs.ErrNotExist) {
				return fs.SkipAll
			}
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if !d.Type().IsRegular() || strings.HasPrefix(d.Name(), tmpPrefix) ||
			!strings.HasPrefix(p, opts.Prefix) || p <= opts.Token {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return nil // removed during the walk
		}
		infos = append(infos, ObjectInfo{Key: p, Size: fi.Size(), LastModified: fi.ModTime().UTC()})
		return nil
	})
	return infos, err
}
